package engine

// TODO: interface that Context will implement, Context should be got from pool
// so we alloc Context in heap already !!

import "syscall"

// func that builds resp, engine works only w bytes, no HTTP logic
type buildFunc func(dst []byte) int

// get buf from pool, write response and put it back
// so we don't alloc new bufs for every resp
func WriteBuf(s *Session, cb buildFunc) (int, error) {
	out := bufPool.Get().([]byte)
	out = out[:cap(out)]

	n := cb(out)
	n, err := syscall.Write(s.Fd, out[:n])

	bufPool.Put(out)
	return n, err
}
