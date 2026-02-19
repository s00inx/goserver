package internal

// TODO: error handling (!)
import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

const (
	maxRequestSize = 1<<16 - 1
)

// session for LT epoll
// buf, offset for raw data
type session struct {
	buf    []byte
	offset int

	hbuf [64]header
	req  request
}

// reset session for put it to pool
func (s *session) reset() {
	s.req = request{}
	s.offset = 0
}

// pool for sessions
var sessionPool = sync.Pool{
	New: func() any {
		return &session{buf: make([]byte, maxRequestSize)}
	},
}

// handle request (descriptor -> parser -> router -> handler -> write to fd and close it)
func workerEpoll(epollfd int, jobs chan int, sessions []atomic.Pointer[session]) {
	for fd := range jobs {
		s := sessions[fd].Load() // load pointer atomically so we don't get invalid ptr
		if s == nil {
			// get new session from pool
			newsession := sessionPool.Get().(*session)
			newsession.reset()

			sessions[fd].Store(newsession) // atomically make new session
			s = newsession
		}

		n, err := syscall.Read(fd, s.buf[s.offset:])
		if n > 0 {
			s.offset += n
			for {
				cons, parserr := parseraw(s.buf[:s.offset], s.hbuf[:], &s.req)
				if parserr == nil {
					syscall.Write(fd, []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"))

					rem := s.offset - cons
					if rem > 0 {
						copy(s.buf, s.buf[cons:s.offset])
					}
					s.offset = rem

					if s.offset == 0 {
						break
					}
					continue
				} else if errors.Is(parserr, errIncomplete) {
					break
				} else {
					err = parserr
					break
				}
			}
		}

		if (err != nil && err != syscall.EAGAIN) || n == 0 || s.offset > maxRequestSize {
			sessions[fd].Store(nil) // atomically zeroing our ptr to session

			// clearing session before put it to pool
			s.reset()

			sessionPool.Put(s)
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
func startWorkerPool(jobs chan int, epollfd int) {
	// get r limit (means max count of descriptors)
	rlim := syscall.Rlimit{}
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)

	sessions := make([]atomic.Pointer[session], rlim.Max)
	// i use atomic pointer here bc i need atomic access to ptr

	numWorkers := runtime.NumCPU()
	for range numWorkers {
		go workerEpoll(epollfd, jobs, sessions)
	}
}
