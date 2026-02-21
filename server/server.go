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

	handler1 := func(w *router.ResponseWriter, r *router.Request) {
		w.Send(200, []byte{}, []byte("hello world"))
	}

	// handler2 := func(w *router.ResponseWriter, r *router.Request) {
	// 	fmt.Println("Handler 2")
	// }

	srv.R.Get("/h", handler1)

	parseFunc := func(s *engine.Session) {
		onReq := func(s *engine.Session, buf []byte) {
			h := srv.R.Serve(&s.Req)

			if h != nil {
				req := &router.Request{
					Raw: &s.Req,
				}

				w := &router.ResponseWriter{
					Session: s,
				}
				h(w, req)
			} else {
				fmt.Println("error")
			}
		}

		srv.prs.Parse(s, onReq)
	}

	engine.StartEpoll(addr, port, parseFunc)
}
