// listening socket conf

package internal

import "syscall"

// create new socket, bind and start listening
func listenSocket(addr [4]byte, port int) (int, error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0) // make new stream socket (this means tcp)
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
