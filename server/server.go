package server

import (
	"io"
	"sync"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
	"github.com/s00inx/goserver/server/router"
)

// Используем алиасы типов (Type Aliasing), чтобы main не импортировал router
type Context = router.Context
type Handler = router.Handler

type Server struct {
	R      *router.HTTPRouter
	parser protocol.HTTPParser
	engine engine.Engine
}

var ctxPool = sync.Pool{
	New: func() any {
		return &router.Context{}
	},
}

func New() *Server {
	return &Server{
		R:      router.NewHTTPRouter(),
		parser: protocol.HTTPParser{},
		engine: engine.Engine{},
	}
}

func (srv *Server) Get(path string, h Handler)  { srv.R.Get(path, h) }
func (srv *Server) Post(path string, h Handler) { srv.R.Post(path, h) }
func (srv *Server) Use(mw Handler)              { srv.R.Use(mw) }
func (srv *Server) Group(prefix string) *Group {
	return &Group{rg: srv.R.Group(prefix)}
}

func (srv *Server) Run(addr [4]byte, port int) error {
	parseFunc := func(s *engine.Session) (bool, error) {
		onReq := func(s *engine.Session, buf []byte) {
			handlers := srv.R.Serve(s)
			c := ctxPool.Get().(*router.Context)
			c.Reset(s, handlers)

			if handlers != nil {
				c.Next()
			} else {
				c.Send404()
			}
			ctxPool.Put(c)
		}
		return srv.parser.Parse(s, onReq)
	}

	return srv.engine.StartEpoll(addr, port, parseFunc)
}

func (srv *Server) Stop(out *io.Writer) {
	srv.engine.StopServer(out)
}

type Group struct {
	rg *router.RouteGroup
}

func (g *Group) Get(path string, h Handler)  { g.rg.Get(path, h) }
func (g *Group) Post(path string, h Handler) { g.rg.Post(path, h) }
func (g *Group) Group(prefix string) *Group  { return &Group{rg: g.rg.Group(prefix)} }
