package engine

import (
	"syscall"
)

var (
	res404 = []byte("HTTP/1.1 404 Not Found\r\nContent-Length: 9\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\nNot Found")
	res500 = []byte("HTTP/1.1 500 Internal Server Error\r\nContent-Length: 21\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\nInternal Server Error")
)

// callback func for build a response
type buildFunc func(dst []byte) int

// get buf from pool, write response and put it back
// so we don't alloc new bufs for every resp
func WriteBuf(s *Session, cb buildFunc) (int, error) {
	rawo := bufPool.Get()
	out := rawo.([]byte)
	out = out[:cap(out)]

	n := cb(out)
	n, err := syscall.Write(int(s.Fd), out[:n])

	bufPool.Put(rawo)
	return n, err
}

func Write404(s *Session) {
	syscall.Write(int(s.Fd), res404)
}

func Write500(s *Session) {
	syscall.Write(int(s.Fd), res500)
}
