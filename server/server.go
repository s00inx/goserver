package server

import (
	"sync"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
	"github.com/s00inx/goserver/server/router"
)

// TEMPORARY
type Context struct {
	router.Context
}

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

func New() *Server {
	return &Server{
		R:   router.NewHTTPRouter(),
		prs: protocol.HTTPParser{},
	}
}

func (srv *Server) Run(addr [4]byte, port int) error {
	parseFunc := func(s *engine.Session) (bool, error) {
		c := ctxPool.Get().(*router.Context)

		onReq := func(s *engine.Session, buf []byte) {
			h := srv.R.Serve(s)
			c.Reset(s, h)

			if h != nil {
				h[0](c)
			} else {
				c.Send404()
			}
			ctxPool.Put(c)
		}
		return srv.prs.Parse(s, onReq)
	}

	return engine.StartEpoll(addr, port, parseFunc)
}
