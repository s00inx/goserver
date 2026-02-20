package goserver

import (
	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
	"github.com/s00inx/goserver/server/router"
)

type Server struct {
	router router.HTTPRouter
	parser protocol.HTTPParser
}

func (s *Server) Run() {
	addr := [4]byte{127, 0, 0, 1}
	port := 8080
	engine.StartEpoll(addr, port, func(fd int, sess *engine.Session) {
		if err := s.parser.Parse(fd, sess, func(fd int, req *engine.RawRequest, buf []byte) {

		}); err != nil {
			return
		}
	})
}
