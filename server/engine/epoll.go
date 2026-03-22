// file with epoll settings and socket creating
// only low level epoll and socket functional
package engine

import (
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
)

// engine struct for storing session state (mainly for graceful shutdown)
type Engine struct {
	lsfd, epollfd int
	sessions      []atomic.Pointer[Session]
	jobsarr       []chan int
}

const (
	backlog   = 256 // backlog for listening
	maxEvents = 256
)

// callback func for handling raw data from socket,
// fd is socket descriptor, and s is Session related to this descriptor
type handleConn func(s *Session) (bool, error)

// starting our server;
// should be called from server.go;
// arguments: address, port and handle conn func (do w socket)
func (e *Engine) StartEpoll(addr [4]byte, port int, cb handleConn) error {
	fd, err := listenSocket(addr, port)
	if err != nil {
		return err
	}
	defer syscall.Close(fd)
	e.lsfd = fd

	// creating new epoll instance
	epollfd, _ := syscall.EpollCreate1(0)
	e.epollfd = epollfd
	// register listening socket to epoll
	syscall.EpollCtl(epollfd, syscall.EPOLL_CTL_ADD, fd, &syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(fd),
	})

	// get r limit (means max count of descriptors)
	rlim := syscall.Rlimit{}
	syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)

	// i use atomic pointer here bc i need atomic access to ptr
	Sessions := make([]atomic.Pointer[Session], rlim.Cur)
	e.sessions = Sessions

	// эта реализация предсказуема и хороша, но для большей производительности можно использовать арену структур,
	// сейчас проблема в том, что сессии лежат в разнвх областях кучи, то есть если она фрагментированна, процессор каждый раз промахивается по кешу
	// если сделать арену структур, то данные будут ложится в кеш, Hardware Prefetcher загрузит текущую сессию.
	// данные будут выделяться в куче линией вначале, это будет занимать больше места (чем 8 байт на указатель),
	// но обеспечат максимальный перформанс

	numworkers := runtime.NumCPU()
	jobs := make([]chan int, numworkers)
	for i := range numworkers {
		jobs[i] = make(chan int, 1<<10)
		go workerEpoll(epollfd, jobs[i], Sessions, cb)
	}
	e.jobsarr = jobs
	events := make([]syscall.EpollEvent, maxEvents)

	e.UpdateDate()

	ticker := time.NewTicker(time.Second)

	// я создаю один глобальный тикер при инициализации еполла
	// такой подход выбран чтобы привязать таймер к конкретному воркеру и конкретному потоку, не запуская отдельную горутину под него

	for {
		select {
		case <-ticker.C:
			e.UpdateDate()
			for i := range jobs {
				jobs[i] <- -1
			}

			// e.PrintStats()
		default:
			// number of events to accept
			n, err := syscall.EpollWait(epollfd, events, -1)
			if err != nil {
				atomic.AddUint64(&Stats.EpollErrors, 1)
				continue
			}

			for i := range n {
				efd := int(events[i].Fd) // current event descriptor

				if efd == fd {
					nfd, _, err := syscall.Accept(fd) // new descriptor for new client
					if err != nil {
						atomic.AddUint64(&Stats.EpollErrors, 1)
					}
					syscall.SetNonblock(nfd, true)

					syscall.EpollCtl(epollfd, syscall.EPOLL_CTL_ADD, nfd, // adding new descriptor to epoll
						&syscall.EpollEvent{
							Events: syscall.EPOLLIN | syscall.EPOLLONESHOT,
							Fd:     int32(nfd),
						})
					syscall.SetsockoptInt(nfd, syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1)
					atomic.AddInt64(&Stats.ActiveConn, 1)
				} else {
					jobs[efd%numworkers] <- efd
				}
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
