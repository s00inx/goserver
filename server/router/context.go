// context is ResponseWriter + HttpRequest !
package router

import (
	"bytes"
	"io"
	"unsafe"

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

	handlers []Handler
	chindex  int
}

// unsafe AREA (i use unsafe bc []byte is not comfortable for business logic, so we need to convert String to Bytes w zero alloc)

// convert string to bytes (NOTE: this method is READ-ONLY, so do not change slice and string) !!
func S2Bytes(s string) []byte {
	if s == "" {
		return nil
	}

	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// convert bytes to string
// (NOTE: this method is READ-ONLY, so do not change slice and string) !!
func B2String(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
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

// get full unparsed url query
func (c *Context) Query() []byte {
	return c.Session.Req.RawQuery.AsBuf(c.Session)
}

// get url query value by key
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

// get all headers to buf
func (c *Context) Headers(buf []engine.Header) []engine.Header {
	count := int(c.Session.Req.Hcount)

	for i := range min(cap(buf), count) {
		h := c.Session.Hbuf[i]
		buf[i] = engine.Header{
			Key: h.Key.AsBuf(c.Session),
			Val: h.Val.AsBuf(c.Session),
		}
	}

	return buf[:count]
}

// get header by key
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

// Get request cookies
func (c *Context) GetCookies() []byte {
	return c.Header(S2Bytes("Cookies"))
}

// context body as []byte (0 alloc)
func (c *Context) Body() []byte {
	return c.Session.Req.Body.AsBuf(c.Session)
}

// body as io.Reader, for convenience (alloc so use it carefully)
func (c *Context) BodyReader() io.Reader {
	return bytes.NewReader(c.Session.Req.Body.AsBuf(c.Session))
}

// reset context for pool !!
func (c *Context) Reset(s *engine.Session, handlers []Handler) {
	c.Session = s
	c.hC = 0
	c.code = 200

	c.chindex = 0
	c.handlers = handlers
}

// ! Context as Response Writer (setters)
// helper func to send resp via engine method
func (c *Context) sendresp(co int, h []engine.Header, b []byte) {
	engine.WriteBuf(c.Session, func(dst []byte) int {
		return protocol.BuildResp(co, h, b, dst)
	})
}

// set resp code
func (c *Context) SetCode(code int) {
	c.code = code
}

// set resp header with int val without overheads
func (c *Context) SetHeaderInt(key []byte, val int) {
	if c.hC > len(c.resH) {
		return
	}
	c.resH[c.hC] = engine.Header{Key: key, Val: protocol.IntToByte(val)}
	c.hC++
}

// set header with []byte key and val
func (c *Context) SetHeader(key, val []byte) {
	if c.hC > len(c.resH) {
		return
	}
	c.resH[c.hC] = engine.Header{Key: key, Val: val}
	c.hC++
}

// send resp direct with 0 alloc,
// you should set headers by SetHeader func
func (c *Context) SendDirect(code int, body []byte) {
	c.SetHeader([]byte("Content-Length"), protocol.IntToByte(len(body)))
	c.sendresp(code, c.resH[:], body)
}

func (c *Context) SendWithBody(body []byte) {
	c.sendresp(c.code, c.resH[:], body)
}

// Middleware functional !!
func (c *Context) Next() {
	c.chindex++
	if c.chindex < len(c.handlers) {
		c.handlers[c.chindex](c)
	}
}
