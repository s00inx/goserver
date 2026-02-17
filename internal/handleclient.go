package internal

import (
	"bytes"
	"runtime"
	"syscall"
)

// session for LT epoll
// buf, offset for raw data, request for 0 alloc
type session struct {
	buf    []byte
	offset int

	req request
}

// req bytes -> resp bytes logic (for testing)
func processConn(raw []byte, req *request) ([]byte, error) {
	buf := make([]header, 0, 4096) // should use pool

	_, err := parseRequest(raw, buf, req)
	if err != nil {
		return nil, err
	}

	return []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"), nil
}

// handle request (descriptor -> parser -> router -> handler -> write to fd n close it )
func handle(epollfd int, jobs chan int, sessions []*session) {
	for fd := range jobs {
		s := sessions[fd]
		if s == nil {
			s = &session{buf: make([]byte, 8192)}
			sessions[fd] = s
		}

		n, err := syscall.Read(fd, s.buf[s.offset:])
		if n > 0 {
			s.offset += n

			if bytes.Contains(s.buf[:s.offset], []byte("\r\n\r\n")) {
				resp, _ := processConn(s.buf[:s.offset], &s.req)
				syscall.Write(fd, resp)

				s.offset = 0
			}
		}

		if (err != nil && err != syscall.EAGAIN) || n == 0 {
			syscall.Close(fd)
			sessions[fd] = nil
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
	numWorkers := runtime.NumCPU()
	sessions := make([]*session, 1<<16-1)
	for range numWorkers {
		go handle(epollfd, jobs, sessions)
	}
}
