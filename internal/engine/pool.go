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
	maxRequestSize = 1<<16 - 1
)

// request struct
// all slices are pointers to Buf for zero-copy
type Request struct {
	Method   []byte
	Path     []byte
	Protocol []byte

	Headers []Header
	Body    []byte
}

// header
// key and val refers to raw data slice
type Header struct {
	Key, Val []byte
}

// session is an arena for pre-allocated request data
// buf, offset for raw data, hbuf and req is pre-allocated buffer for headers and request struct from pool
type Session struct {
	Buf    []byte
	Offset int

	Hbuf [64]Header
	Req  Request
}

// parser from protocol layer
// so we need interface to use them
type httpParser interface {
	Parse(fd int, s *Session) error
}

// reset session for put it to pool
func (s *Session) reset() {
	s.Req = Request{}
	s.Offset = 0
}

// pool for sessions
var SessionPool = sync.Pool{
	New: func() any {
		return &Session{Buf: make([]byte, maxRequestSize)}
	},
}

// handle request // fd -> parser -> router -> handler -> write & close
func workerEpoll(epollfd int, jobs chan int, Sessions []atomic.Pointer[Session], p httpParser) {
	for fd := range jobs {
		s := Sessions[fd].Load() // load pointer atomically so we don't get invalid ptr
		if s == nil {
			// get new session from pool
			newsession := SessionPool.Get().(*Session)
			newsession.reset()

			Sessions[fd].Store(newsession) // atomically make new session
			s = newsession
		}

		n, err := syscall.Read(fd, s.Buf[s.Offset:])
		if n > 0 {
			s.Offset += n

			p.Parse(fd, s)
		}

		if (err != nil && err != syscall.EAGAIN) || n == 0 || s.Offset > maxRequestSize {
			Sessions[fd].Store(nil) // atomically zeroing our ptr to session

			// clearing session before put it to pool
			s.reset()

			SessionPool.Put(s)
			syscall.Close(fd) // closing socket AFTER putting it to pool
			continue
		}

		ev := syscall.EpollEvent{
			Events: syscall.EPOLLIN | syscall.EPOLLONESHOT,
			Fd:     int32(fd),
		}
		syscall.EpollCtl(epollfd, syscall.EPOLL_CTL_MOD, fd, &ev)
	}
}

// start simple worker pool for handling a request
func startWorkerPool(jobs chan int, epollfd int, p httpParser) {
	// get r limit (means max count of descriptors)
	rlim := syscall.Rlimit{}
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)

	Sessions := make([]atomic.Pointer[Session], rlim.Max)
	// i use atomic pointer here bc i need atomic access to ptr

	numWorkers := runtime.NumCPU()
	for range numWorkers {
		go workerEpoll(epollfd, jobs, Sessions, p)
	}
}
