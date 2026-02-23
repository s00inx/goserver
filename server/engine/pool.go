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
	maxRawSize = 1<<16 - 1
)

// request struct, raw because it refers to bytes so we can't use it in user scope, we have Request for it
// all slices are pointers to session Buf for zero-copy
type RawRequest struct {
	Method   []byte // http method
	Protocol []byte // proto

	Path     []byte  // url
	Params   []Param // url params
	Pcount   int     // url param count
	RawQuery []byte  // url query (? ...) raw bc i don't parse it if not needed

	Headers []Header // req headers
	Body    []byte   // req body
}

// header
// key and val refers to raw data slice
type Header struct {
	Key, Val []byte
}

// path parameters (:id, etc.)
type Param struct {
	Key, Val []byte
}

// session is an arena for pre-allocated data
// it manages buffers and fd for HTTPRequest, session is atomical instance for 1 socket fd !
// buf, offset for raw data, hbuf and req is pre-allocated buffer for headers and RawRequest struct from pool
type Session struct {
	Fd     int    // 8 byte
	Buf    []byte // depends on maxRequestSize ; buf sets off only when session need it, see workerEpoll func
	Offset int    // 8 byte

	Hbuf [64]Header // buf for headers, 64 * 24 on start
	Pbuf [16]Param  // buf for path parameters, 64 * 24 on start
	Req  RawRequest // 24 bytes (slice view) bc it is window to s.Buf --^
}

// reset session for put it to pool
func (s *Session) reset() {
	s.Fd = 0
	s.Offset = 0

	s.Req = RawRequest{}
	s.Req.Pcount = 0
}

// pool for sessions
var (
	// bufPool for sessions buffers
	bufPool = sync.Pool{
		New: func() any {
			return make([]byte, maxRawSize)
		},
	}

	// pool for sessions
	sessionPool = sync.Pool{
		New: func() any {
			return &Session{}
		},
	}
)

// handle RawRequest // fd -> parser -> router -> handler -> write & close
func workerEpoll(epollfd int, jobs chan int, Sessions []atomic.Pointer[Session], cb handleConn) {
	for fd := range jobs {
		s := Sessions[fd].Load() // load pointer atomically so we don't get invalid ptr
		if s == nil {
			// get new session from pool
			newsession := sessionPool.Get().(*Session)
			newsession.reset()
			newsession.Fd = fd

			Sessions[fd].Store(newsession) // atomically make new session
			s = newsession
		}

		// give buffer to session only when needed
		// it is useful when we have many keep-alive conns thst store bufs but not working
		if s.Buf == nil {
			buf := bufPool.Get().([]byte)
			s.Buf = buf[:cap(buf)]
		}

		n, err := syscall.Read(fd, s.Buf[s.Offset:])
		if (err != nil && err != syscall.EAGAIN) || n == 0 || s.Offset > maxRawSize {
			Sessions[fd].Store(nil) // atomically zeroing our ptr to session

			// clearing session before put it to pool
			s.reset()

			sessionPool.Put(s)
			syscall.Close(fd) // closing socket AFTER putting it to pool
			continue
		}

		if n > 0 {
			s.Offset += n
			shouldRelease, _ := cb(s)

			if shouldRelease {
				bufPool.Put(s.Buf)
				s.Buf = nil
				s.Offset = 0
			}
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

	rlim := syscall.Rlimit{}
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)

	Sessions := make([]atomic.Pointer[Session], rlim.Cur)
	// i use atomic pointer here bc i need atomic access to ptr

	numWorkers := runtime.NumCPU()
	for range numWorkers {
		go workerEpoll(epollfd, jobs, Sessions, cb)
	}
}
