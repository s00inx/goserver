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
	return c.Session.Req.Method.AsBuf(c.Session)
}

// get Request path
func (c *Context) Path() []byte {
	return c.Session.Req.Path.AsBuf(c.Session)
}

func (c *Context) Query() []byte {
	return c.Session.Req.RawQuery.AsBuf(c.Session)
}

func (c *Context) QueryGet(key []byte) []byte {
	view := c.Session.Req.RawQuery
	if view.End <= view.St {
		return nil
	}

	q := view.AsBuf(c.Session)

	for len(q) > 0 {
		var pair []byte
		idx := bytes.IndexByte(q, '&')

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
	return c.Session.Req.Protocol.AsBuf(c.Session)
}

func (c *Context) Params() []engine.Param {
	count := int(c.Session.Req.Pcount)
	if count == 0 {
		return nil
	}

	return c.Session.Pbuf[:count]
}

func (c *Context) Param(key []byte) []byte {
	count := int(c.Session.Req.Pcount)

	for i := range count {
		p := &c.Session.Pbuf[i]

		if bytes.Equal(p.Key, key) {
			return p.Val.AsBuf(c.Session)
		}
	}
	return nil
}

func (c *Context) ParamCount() int {
	return int(c.Session.Req.Pcount)
}

func (c *Context) Headers() []engine.Header {
	count := int(c.Session.Req.Hcount)
	res := make([]engine.Header, count)

	for i := range count {
		h := c.Session.Hbuf[i]
		res[i] = engine.Header{
			Key: h.Key.AsBuf(c.Session),
			Val: h.Val.AsBuf(c.Session),
		}
	}

	return res
}

func (c *Context) Header(key []byte) []byte {
	count := int(c.Session.Req.Hcount)

	for i := range count {
		h := &c.Session.Hbuf[i]
		if bytes.Equal(h.Key.AsBuf(c.Session), key) {
			return h.Val.AsBuf(c.Session)
		}
	}
	return nil
}

func (c *Context) Body() []byte {
	return c.Session.Req.Body.AsBuf(c.Session)
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
