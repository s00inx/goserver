// session management and worker logic
package engine

import (
	"runtime"
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
func workerEpoll(epollfd int, jobs chan int, Sessions []atomic.Pointer[Session], tw *TimerWheel, cb handleConn) {
	for fd := range jobs {
		s := Sessions[fd].Load() // load pointer atomically so we don't get invalid ptr
		if s == nil {
			nsRaw := sessionPool.Get()
			ns := nsRaw.(*Session)
			ns.Reset()
			ns.Fd = uint32(fd)
			ns.raw = nsRaw

			if Sessions[fd].CompareAndSwap(nil, ns) {
				s = ns
				// tw.Update(s)
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

// start simple worker pool for handling a RawRequest
func startWorkerPool(jobs chan int, epollfd int, cb handleConn) {
	// get r limit (means max count of descriptors)

	rlim := syscall.Rlimit{}
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)

	Sessions := make([]atomic.Pointer[Session], rlim.Cur)
	// i use atomic pointer here bc i need atomic access to ptr

	// эта реализация предсказуема и хороша, но для большей производительности можно использовать арену структур,
	// сейчас проблема в том, что сессии лежат в разнвх областях кучи, то есть если она фрагментированна, процессор каждый раз промахивается по кешу
	// если сделать арену структур, то данные будут ложится в кеш, Hardware Prefetcher загрузит текущую сессию.
	// данные будут выделяться в куче линией вначале, это будет занимать больше места (чем 8 байт на указатель),
	// но обеспечат максимальный перформанс

	// tw := TimerWheel{
	// 	mask: 3,
	// }
	// go tw.StartKiller(Sessions)

	numWorkers := runtime.NumCPU()
	for range numWorkers {
		go workerEpoll(epollfd, jobs, Sessions, nil, cb)
	}
}
