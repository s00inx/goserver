package router

import (
	"syscall"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
)

// response writer instance
type ResponseWriter struct {
	Session *engine.Session
	wrote   bool
}

// send http resp with code, headers and body
func (w *ResponseWriter) Send(code int, headers []byte, body []byte) {
	dst := make([]byte, 4096) // TEMPORARY!!!!! will be pre-alloc buf from session
	protocol.BuildResp(code, body, dst)

	syscall.Write(w.Session.Fd, dst)
}
