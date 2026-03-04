package server

import (
	"io"
	"sync"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
	"github.com/s00inx/goserver/server/router"
)

// TEMPORARY
type Context struct {
	router.Context
}

type Handler struct {
	router.Handler
}

type Server struct {
	R *router.HTTPRouter

	parser protocol.HTTPParser
	engine engine.Engine
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
		R:      router.NewHTTPRouter(),
		parser: protocol.HTTPParser{},
		engine: engine.Engine{},
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
		return srv.parser.Parse(s, onReq)
	}

	return srv.engine.StartEpoll(addr, port, parseFunc)
}

// basically there is alloc (interface conversion), but since this is one-time operation it won't affect runtime
func (srv *Server) Stop(out *io.Writer) {
	srv.engine.StopServer(out)
}
