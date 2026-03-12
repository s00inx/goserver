package main

import (
	"os"
	"os/signal"
	"syscall"

	srv "github.com/s00inx/goserver/server"
)

func main() {
	s := srv.New()

	handler := func(c *srv.Context) {
		c.SendDirect(200, []byte("hello world!"))
	}

	s.R.Get("/h", handler)

	hg := s.Group("/h")
	hg.Get("/1", handler)

	go func() {
		s.Run([4]byte{127, 0, 0, 1}, 8080)
	}()

	stop := make(chan os.Signal, 2)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	s.Stop(nil)
}
