// session management and worker logic
package engine

// TODO: error handling (!)
// TODO: optimize session size, replace []byte (24 B) -> 2 uint16 (4 B)
import (
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

const (
	maxRawSize = 1<<16 - 1
)

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
			nsRaw := sessionPool.Get()
			ns := nsRaw.(*Session)

			ns.Reset()
			ns.Fd = uint32(fd)

			Sessions[fd].Store(ns) // atomically make new session
			s = ns
			s.raw = nsRaw
		}

		// give buffer to session only when needed
		// it is useful when we have many keep-alive conns thst store bufs but not working
		if s.Buf == nil {
			bufraw := bufPool.Get()
			buf := bufraw.([]byte)

			s.bufraw = bufraw
			s.Buf = buf[:cap(buf)]
		}

		n, err := syscall.Read(fd, s.Buf[s.Offset:])
		if (err != nil && err != syscall.EAGAIN) || n == 0 || s.Offset > maxRawSize {
			Sessions[fd].Store(nil) // atomically zeroing our ptr to session

			// clearing session before put it to pool
			s.Reset()

			sessionPool.Put(s.raw)
			syscall.Close(fd) // closing socket AFTER putting it to pool
			continue
		}

		if n > 0 {
			s.Offset += uint32(n)
			shouldRelease, _ := cb(s)

			if shouldRelease {
				bufPool.Put(s.bufraw)
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
