package protocol

import (
	"testing"

	"github.com/s00inx/goserver/server/engine"
)

func BenchmarkParse(b *testing.B) {
	p := &HTTPParser{}
	raw := []byte("POST /very/long/path/for/testing/purposes HTTP/1.1\r\n" +
		"Host: localhost:8080\r\n" +
		"User-Agent: goserver-benchmark\r\n" +
		"Content-Length: 18\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		"{\"key\":\"value_123\"}")

	hbuf := make([]engine.Header, 64)
	req := &engine.RawRequest{}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = p.parseRaw(raw, hbuf, req)
		req.Headers = hbuf[:0]
	}
}
