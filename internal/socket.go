package internal

import (
	"syscall"
)

const (
	backlog = 16 // backlog for listening
)

// create new socket, bind and start listening
func listenSocket(addr [4]byte, port int) (int, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return -1, err
	}

	if err := syscall.Bind(fd, &syscall.SockaddrInet4{
		Port: port,
		Addr: addr,
	}); err != nil {
		return -1, err
	}
	if err := syscall.Listen(fd, backlog); err != nil {
		return -1, err
	}

	// log.Printf("new socket started on %d:%d, fd = %d", addr, port, fd)
	return fd, nil
}

// create new epoll
func EpollRecv(addr [4]byte, port int) error {
	fd, err := listenSocket(addr, port)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	epollfd, _ := syscall.EpollCreate1(0) // creating new epoll object
	syscall.EpollCtl(epollfd, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(fd),
	}) // adding event w peer socket descriptor

	jobs := make(chan int, 1024)
	startWorkerPool(jobs, epollfd)

	events := make([]syscall.EpollEvent, 64)
	for {
		// number of events to accept
		n, err := syscall.EpollWait(epollfd, events, -1)
		if err != nil {
			continue
		}

		for i := range n {
			efd := int(events[i].Fd) // current event descriptor

			if efd == fd {
				nfd, _, _ := syscall.Accept(fd) // new descriptor for new client
				syscall.SetNonblock(nfd, true)

				syscall.EpollCtl(epollfd, syscall.EPOLL_CTL_ADD, nfd, // adding new descriptor to epoll
					&syscall.EpollEvent{
						Events: syscall.EPOLLIN | syscall.EPOLLONESHOT,
						Fd:     int32(nfd),
					})
				// log.Printf("new client connected: %d\n", nfd)
			} else {
				jobs <- efd
			}
		}
	}
}
