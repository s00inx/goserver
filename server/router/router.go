package router

import (
	"github.com/s00inx/goserver/server/engine"
)

type HTTPRouter struct {
	treeroot Node
}

// init a new router
func NewHTTPRouter() *HTTPRouter {
	return &HTTPRouter{
		treeroot: InitRoot(),
	}
}

func (r *HTTPRouter) Route(path []byte, h Handler) {
	r.treeroot.Insert(path, h)
}

func (r *HTTPRouter) Serve(rreq *engine.RawRequest) Handler {
	return r.treeroot.Match(rreq.Path, rreq)
}
