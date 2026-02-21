// file with epoll settings and socket creating
// only low level epoll and socket functional
package engine

import (
	"syscall"
)

const (
	backlog   = 16 // backlog for listening
	maxEvents = 128
)

// callback func for handling raw data from socket,
// fd is socket descriptor, and s is Session related to this descriptor
type handleConn func(s *Session)

// starting our server;
// should be called from server.go;
// arguments: address, port and handle conn func (do w socket)
func StartEpoll(addr [4]byte, port int, cb handleConn) error {
	fd, err := listenSocket(addr, port)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)

	// creating new epoll instance
	epollfd, _ := syscall.EpollCreate1(0)

	// register listening socket to epoll
	syscall.EpollCtl(epollfd, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(fd),
	})

	jobs := make(chan int, 1024)
	startWorkerPool(jobs, epollfd, cb)

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

// create new socket, bind and start listening
func listenSocket(addr [4]byte, port int) (int, error) {
	// SOCK_STREAM = TCP
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return -1, err
	}

	if err := syscall.Bind(fd, &syscall.SockaddrInet4{ // bind socket to addr:port
		Port: port,
		Addr: addr,
	}); err != nil {
		return -1, err
	}
	if err := syscall.Listen(fd, backlog); err != nil { // start listening on addr:port
		return -1, err
	}

	// log.Printf("new socket started on %d:%d, fd = %d", addr, port, fd)
	return fd, nil
}
