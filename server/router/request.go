package router

import "github.com/s00inx/goserver/server/engine"

type Request struct {
	fd  int
	raw *engine.RawRequest
}

func (r *Request) Path() {}

// ...

func (r *Request) Send(code int, message string) {

}
