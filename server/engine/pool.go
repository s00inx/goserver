// session management and worker logic
package engine

// TODO: error handling (!)
import (
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

const (
	maxRawRequestSize = 1<<16 - 1
)

// request struct, raw because it refers to bytes so we can't use it in user scope, we have Request for it
// all slices are pointers to session Buf for zero-copy
type RawRequest struct {
	Method   []byte
	Path     []byte
	Protocol []byte

	Headers []Header
	P       []Param
	Body    []byte

	Pcount int
}

// header
// key and val refers to raw data slice
type Header struct {
	Key, Val []byte
}

type Param struct {
	Key, Val []byte
}

// session is an arena for pre-allocated RawRequest data
// buf, offset for raw data, hbuf and req is pre-allocated buffer for headers and RawRequest struct from pool
type Session struct {
	Buf    []byte // dep. on maxRequestSize its 2**16 - 1 byte
	Offset int    // 4 byte

	Hbuf [64]Header // 64 * ?? byte
	Req  RawRequest // 24 bytes (slice ptr, len and cap) bc it is window to s.Buf --^
}

// reset session for put it to pool
func (s *Session) reset() {
	s.Req = RawRequest{}
	s.Offset = 0
	s.Req.P = []Param{}
	s.Req.Pcount = 0
}

// pool for sessions
var sessionPool = sync.Pool{
	New: func() any {
		return &Session{Buf: make([]byte, maxRawRequestSize)}
	},
}

// handle RawRequest // fd -> parser -> router -> handler -> write & close
func workerEpoll(epollfd int, jobs chan int, Sessions []atomic.Pointer[Session], cb handleConn) {
	for fd := range jobs {
		s := Sessions[fd].Load() // load pointer atomically so we don't get invalid ptr
		if s == nil {
			// get new session from pool
			newsession := sessionPool.Get().(*Session)
			newsession.reset()

			Sessions[fd].Store(newsession) // atomically make new session
			s = newsession
		}

		n, err := syscall.Read(fd, s.Buf[s.Offset:])
		if (err != nil && err != syscall.EAGAIN) || n == 0 || s.Offset > maxRawRequestSize {
			Sessions[fd].Store(nil) // atomically zeroing our ptr to session

			// clearing session before put it to pool
			s.reset()

			sessionPool.Put(s)
			syscall.Close(fd) // closing socket AFTER putting it to pool
			continue
		}
		if n > 0 {
			s.Offset += n
			cb(fd, s)
		}

		ev := syscall.EpollEvent{
			Events: syscall.EPOLLIN | syscall.EPOLLONESHOT,
			Fd:     int32(fd),
		}
		syscall.EpollCtl(epollfd, syscall.EPOLL_CTL_MOD, fd, &ev)
	}
}

// start simple worker pool for handling a RawRequest
func startWorkerPool(jobs chan int, epollfd int, cb handleConn) {
	// get r limit (means max count of descriptors)
	// this is unsafe (atm) bc rlimoit can be very large so we have a lot of unused pre-allocated memory
	rlim := syscall.Rlimit{}
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)

	Sessions := make([]atomic.Pointer[Session], rlim.Cur)
	// i use atomic pointer here bc i need atomic access to ptr

	numWorkers := runtime.NumCPU()
	for range numWorkers {
		go workerEpoll(epollfd, jobs, Sessions, cb)
	}
}
