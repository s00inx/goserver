package main

import (
	srv "github.com/s00inx/goserver/server"
	"github.com/s00inx/goserver/server/router"
)

func main() {
	s := srv.New()

	handler := func(c *router.Context) {
		c.SendDirect(200, []byte("hello world!"))
	}

	s.R.Get("/h", handler)

	s.Run([4]byte{127, 0, 0, 1}, 8080)
}
