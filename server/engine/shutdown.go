package engine

import (
	"io"
	"os"
	"syscall"
	"time"
)

// stops the server and write logs to stdout (default os.Stdout)
func (e *Engine) StopServer(stdout *io.Writer) {
	var out io.Writer
	if stdout == nil {
		out = os.Stdout
	} else {
		out = *stdout
	}

	syscall.Close(e.lsfd)
	out.Write([]byte("closing listening socket...\n"))

	for _, ch := range e.jobsarr {
		close(ch)
	}
	out.Write([]byte("closing worker channels...\n"))

	time.Sleep(time.Millisecond * 500)
	syscall.Close(e.epollfd)
	out.Write([]byte("closing epoll descpiptor...\n"))

	for i := range e.sessions {
		se := e.sessions[i].Swap(nil)

		if se != nil {
			syscall.Close(int(se.Fd))
		}
	}
	out.Write([]byte("! server is down...\n"))
}
