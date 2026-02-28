package server

import (
	"fmt"
	"sync"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
	"github.com/s00inx/goserver/server/router"
)

type Server struct {
	R   *router.HTTPRouter
	prs protocol.HTTPParser
}

var (
	ctxPool = sync.Pool{
		New: func() any {
			return &router.Context{}
		},
	}
)

func Test() {
	addr, port := [4]byte{127, 0, 0, 1}, 8080
	srv := Server{
		R:   router.NewHTTPRouter(),
		prs: protocol.HTTPParser{},
	}

	handler1 := func(c *router.Context) {
		c.SendDirect(200, []byte("hello"))
	}

	srv.R.Get("/h", handler1)

	parseFunc := func(s *engine.Session) (bool, error) {
		onReq := func(s *engine.Session, buf []byte) {
			h := srv.R.Serve(s)

			if h != nil {
				ctxPtr := ctxPool.Get()
				c := ctxPtr.(router.Context)

				c.Reset(s, []router.Handler{})

				h(&c)
			} else {
				fmt.Println("error")
			}
		}

		return srv.prs.Parse(s, onReq)
	}

	engine.StartEpoll(addr, port, parseFunc)
}
