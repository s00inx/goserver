package internal

import (
	"fmt"
	"syscall"
)

// type resp for test
const resp = "HTTP/1.1 200 OK\r\n" +
	"Content-Length: 12\r\n" +
	"Content-Type: text/plain\r\n" +
	"\r\n" +
	"Hello World!"

func Test(ip [4]byte, port int) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		fmt.Print(err)
		return
	}
	defer syscall.Close(fd)

	sa := &syscall.SockaddrInet4{
		Port: port,
		Addr: ip,
	}
	if err := syscall.Bind(fd, sa); err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Socket bound into: %d:%d, (fd = %d)\n", ip, port, fd)

	err = syscall.Listen(fd, 5)
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		nfd, _, err := syscall.Accept(fd)
		if err != nil {
			fmt.Print(err)
			return
		}
		go func(fd int) {
			defer syscall.Close(fd)

			syscall.Write(fd, []byte(resp))

			rb := make([]byte, 1024)
			n, _ := syscall.Read(fd, rb)

			fmt.Printf("received %d bytes: %d\n%s", n, fd, rb[:n])
		}(nfd)
	}
}
