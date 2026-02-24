package engine

import (
	"syscall"
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
