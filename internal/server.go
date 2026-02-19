package internal

import (
	"github.com/kfcemployee/goserver/internal/engine"
	"github.com/kfcemployee/goserver/internal/protocol"
)

// starting new server
func Start(addr [4]byte, port int) {
	parser := &protocol.HTTPParser{}
	engine.StartEpoll(addr, port, parser)
}
