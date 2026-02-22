// context is ResponseWriter + HttpRequest !
package router

import (
	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
)

type Handler func(c *Context)

type Context struct {
	Session *engine.Session

	code int
}

// helper func to send resp via engine method
func (c *Context) sendresp(co int, h, b []byte) {
	engine.WriteBuf(c.Session, func(dst []byte) int {
		return protocol.BuildResp(co, h, b, dst)
	})
}

// TEMP !!!
func (c *Context) Send(code int, body []byte) {
	c.sendresp(code, []byte{}, body)
}
