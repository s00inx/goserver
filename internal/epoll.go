// file with epoll settings
package internal

import (
	"syscall"
)

const (
	backlog   = 16 // backlog for listening
	maxEvents = 128
)

// starting our server
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

	events := make([]syscall.EpollEvent, maxEvents)
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
