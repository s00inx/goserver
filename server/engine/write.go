package engine

import "syscall"

// func that builds resp, engine works only w bytes, no HTTP logic
type buildFunc func(dst []byte) int

// get buf from pool, write response and put it back
// so we don't alloc new bufs for every resp
func WriteBuf(s *Session, cb buildFunc) {
	out := bufPool.Get().([]byte)
	out = out[:cap(out)]

	n := cb(out)
	syscall.Write(s.Fd, out[:n])

	bufPool.Put(out)
}
