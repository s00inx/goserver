package router

import (
	"github.com/s00inx/goserver/server/engine"
)

type Handler func(w *ResponseWriter, req *Request)

type Request struct {
	Raw *engine.RawRequest
}

func (r *Request) Path() []byte {
	return r.Raw.Path
}
