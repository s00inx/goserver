// context is ResponseWriter + HttpRequest !
package router

import (
	"bytes"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
)

// handler func signature, it works only with context
type Handler func(c *Context)

type Context struct {
	Session *engine.Session

	code int
	resH [16]engine.Header
	hC   int
}

// !! Context as abstraction upon Session (getters)
// get Request method
func (c *Context) Method() []byte {
	return c.Session.Req.Method
}

// get Request path
func (c *Context) Path() []byte {
	return c.Session.Req.Path
}

func (c *Context) Query() []byte {
	return c.Session.Req.RawQuery
}

func (c *Context) QueryGet(key []byte) []byte {
	q := c.Session.Req.RawQuery
	if len(q) == 0 {
		return nil
	}

	for len(q) > 0 {
		idx := bytes.IndexByte(q, '&')
		var pair []byte
		if idx == -1 {
			pair = q
			q = nil
		} else {
			pair = q[:idx]
			q = q[idx+1:]
		}

		before, after, ok := bytes.Cut(pair, []byte{'='})
		if ok && bytes.Equal(before, key) {
			return after
		}
	}
	return nil
}

func (c *Context) Protocol() []byte {
	return c.Session.Req.Protocol
}

func (c *Context) Params() []engine.Param {
	return c.Session.Req.Params
}

func (c *Context) Param(key []byte) []byte {
	for _, p := range c.Session.Req.Params {
		if bytes.Equal(p.Key, key) {
			return p.Val
		}
	}
	return nil
}

func (c *Context) ParamCount() int {
	return c.Session.Req.Pcount
}

func (c *Context) Headers() []engine.Header {
	return c.Session.Req.Headers
}

func (c *Context) Header(key []byte) []byte {
	for _, h := range c.Session.Req.Headers {
		if bytes.Equal(h.Key, key) {
			return h.Val
		}
	}
	return nil
}

func (c *Context) Body() []byte {
	return c.Session.Req.Body
}

// ! Context as Response Writer (setters)
// helper func to send resp via engine method
func (c *Context) sendresp(co int, h []engine.Header, b []byte) {
	engine.WriteBuf(c.Session, func(dst []byte) int {
		return protocol.BuildResp(co, h, b, dst)
	})
}

func (c *Context) SetCode(code int) {
	c.code = code
}

func (c *Context) SetHeaderInt(key []byte, val int) {
	if c.hC > len(c.resH) {
		return
	}

	c.resH[c.hC] = engine.Header{Key: key, Val: protocol.IntToByte(val)}
	c.hC++
}

func (c *Context) SetHeader(key, val []byte) {
	if c.hC > len(c.resH) {
		return
	}

	c.resH[c.hC] = engine.Header{Key: key, Val: val}
	c.hC++
}

// send resp direct with 0 alloc
// you should set headers by SetHeader func
func (c *Context) SendDirect(code int, body []byte) {
	c.SetHeader([]byte("Content-Length"), protocol.IntToByte(len(body)))
	c.sendresp(code, c.resH[:], body)
}

// TEMP !!!
func (c *Context) Send(code int, body []byte) {
	c.sendresp(code, []engine.Header{}, body)
}
