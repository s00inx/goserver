package server

import (
	"fmt"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
	"github.com/s00inx/goserver/server/router"
)

type Server struct {
	R   *router.HTTPRouter
	prs protocol.HTTPParser
}

func Test() {
	addr, port := [4]byte{127, 0, 0, 1}, 8080
	srv := Server{
		R:   router.NewHTTPRouter(),
		prs: protocol.HTTPParser{},
	}

	handler1 := func(c *router.Context) {
		p := fmt.Appendf(nil, "%s", c.Session.Req.Params)
		c.Send(200, []byte(p))
	}

	srv.R.Get("/:id", handler1)

	parseFunc := func(s *engine.Session) (bool, error) {
		onReq := func(s *engine.Session, buf []byte) {
			h := srv.R.Serve(s)

			if h != nil {
				c := &router.Context{
					Session: s,
				}

				h(c)
			} else {
				fmt.Println("error")
			}
		}

		return srv.prs.Parse(s, onReq)
	}

	engine.StartEpoll(addr, port, parseFunc)
}
