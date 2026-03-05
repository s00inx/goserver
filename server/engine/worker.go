// session management and worker logic
package engine

import (
	"sync"
	"sync/atomic"
	"syscall"
)

const (
	maxRawSize = 1<<16 - 1
)

// pool for sessions
var (
	// bufPool for sessions buffers
	bufPool = sync.Pool{
		New: func() any {
			return make([]byte, maxRawSize)
		},
	}

	// pool for sessions
	sessionPool = sync.Pool{
		New: func() any {
			return &Session{}
		},
	}
)

// handle RawRequest // fd -> parser -> router -> handler -> write & close
func workerEpoll(epollfd int, jobs chan int, Sessions []atomic.Pointer[Session], cb handleConn) {
	tw := NewWheel(20)

	for fd := range jobs {
		if fd == -1 {
			tw.killSharded(Sessions)
			continue
		}

		s := Sessions[fd].Load() // load pointer atomically so we don't get invalid ptr
		if s == nil {
			nsRaw := sessionPool.Get()
			ns := nsRaw.(*Session)
			ns.Reset()
			ns.Fd = uint32(fd)
			ns.raw = nsRaw

			if Sessions[fd].CompareAndSwap(nil, ns) {
				s = ns
				tw.Update(s)
			} else {
				sessionPool.Put(nsRaw)
				s = Sessions[fd].Load()
			}
		}
		if !s.inWork.CompareAndSwap(false, true) {
			continue
		}

		// give buffer to session only when needed
		// it is useful when we have many keep-alive conns thst store bufs but not working
		if s.Buf == nil {
			bufraw := bufPool.Get()
			buf := bufraw.([]byte)

			s.bufraw = bufraw
			s.Buf = buf[:cap(buf)]
		}

		n, err := syscall.Read(fd, s.Buf[s.Offset:])
		if (err != nil && err != syscall.EAGAIN) || n == 0 || s.Offset > maxRawSize {
			if Sessions[fd].CompareAndSwap(s, nil) {

				if s.bufraw != nil {
					bufPool.Put(s.bufraw)
					s.bufraw = nil
					s.Buf = nil
				}

				s.Reset()
				sessionPool.Put(s.raw)
				syscall.Close(fd)
				continue
			}
		}

		if n > 0 {
			tw.Update(s)

			s.Offset += uint32(n)
			shouldRelease, _ := cb(s)

			if shouldRelease {
				bufPool.Put(s.bufraw)
				s.Buf = nil
				s.Offset = 0
			}
		}

		s.inWork.Store(false)

		ev := syscall.EpollEvent{
			Events: syscall.EPOLLIN | syscall.EPOLLONESHOT,
			Fd:     int32(fd),
		}
		syscall.EpollCtl(epollfd, syscall.EPOLL_CTL_MOD, fd, &ev)
	}

}
