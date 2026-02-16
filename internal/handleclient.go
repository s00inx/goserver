package internal

import (
	"log"
	"runtime"
	"syscall"
)

// handle request (descriptor -> parser -> router -> handler -> write to fd n close it )
// will be DONE
func handle(epollfd int, jobs chan int) {
	for fd := range jobs {
		buf := make([]byte, 1024)
		n, err := syscall.Read(fd, buf)

		if n > 0 {
			syscall.Write(fd, []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"))
		}

		if n <= 0 || (err != nil && err != syscall.EAGAIN) {
			syscall.Close(fd)
			continue
		}

		ev := syscall.EpollEvent{
			Events: syscall.EPOLLIN | syscall.EPOLLONESHOT | EPOLLET,
			Fd:     int32(fd),
		}
		syscall.EpollCtl(epollfd, syscall.EPOLL_CTL_MOD, fd, &ev)

		log.Printf("received %d bytes from conn %d\n", n, fd)
	}
}

// start simple worker pool for handling a request
func startWorkerPool(jobs chan int, epollfd int) {
	numWorkers := runtime.NumCPU()
	for range numWorkers {
		go handle(epollfd, jobs)
	}
}
