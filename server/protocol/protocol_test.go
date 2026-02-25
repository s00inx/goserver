package protocol

import (
	"fmt"
	"testing"

	"github.com/s00inx/goserver/server/engine"
)

func BenchmarkBuildResp(b *testing.B) {
	body := []byte("{\"status\":\"ok\",\"message\":\"hello world\"}")
	dst := make([]byte, 1024)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = BuildResp(200, []engine.Header{}, body, dst)
	}
}

func BenchmarkParse(b *testing.B) {
	p := &HTTPParser{}
	raw := []byte("POST /very/long/path/for/testing/purposes HTTP/1.1\r\n" +
		"Host: localhost:8080\r\n" +
		"User-Agent: goserver-benchmark\r\n" +
		"Content-Length: 18\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		"{\"key\":\"value_123\"}")

	hbuf := make([]engine.HeaderView, 64)
	req := &engine.RawRequest{}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = p.parseRaw(raw, hbuf, req)
	}
}

func BenchmarkParseHeavy(b *testing.B) {
	headers := ""
	for i := range 20 {
		headers += fmt.Sprintf("X-Header-%d: value-%d-extra-long-data-for-stress-test\r\n", i, i)
	}
	body := make([]byte, 1024)
	for i := range body {
		body[i] = 'a'
	}

	raw := []byte(fmt.Sprintf("POST /api/v1/resource/update/large HTTP/1.1\r\n"+
		"Host: localhost\r\n"+
		"Content-Length: %d\r\n"+
		"Content-Type: application/octet-stream\r\n"+
		"%s\r\n%s", len(body), headers, body))

	parser := &HTTPParser{}
	s := &engine.Session{
		Buf: make([]byte, 4096),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		s.Offset = uint32(len(raw))
		copy(s.Buf[:], raw)
		s.Req = engine.RawRequest{}

		_, err := parser.Parse(s, func(sess *engine.Session, buf []byte) {
		})

		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestHTTPParser_Parse(t *testing.T) {
	parser := &HTTPParser{}

	t.Run("Simple GET Request", func(t *testing.T) {
		s := &engine.Session{
			Buf: make([]byte, 1024),
		}
		raw := "GET /index.html HTTP/1.1\r\nHost: localhost\r\nUser-Agent: test\r\n\r\n"
		copy(s.Buf, raw)
		s.Offset = uint32(len(raw))

		called := false
		onReq := func(session *engine.Session, buf []byte) {
			called = true
			if string(session.Req.Method.AsBuf(session)) != "GET" {
				t.Errorf("Expected GET, got %s", session.Req.Method.AsBuf(session))
			}
			if session.Req.Hcount != 2 {
				t.Errorf("Expected 2 headers, got %d", session.Req.Hcount)
			}
		}

		_, err := parser.Parse(s, onReq)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !called {
			t.Error("Callback was not called")
		}
	})

	t.Run("POST with Body", func(t *testing.T) {
		s := &engine.Session{
			Buf: make([]byte, 1024),
		}
		raw := "POST /submit HTTP/1.1\r\nContent-Length: 11\r\n\r\nhello world"
		copy(s.Buf, raw)
		s.Offset = uint32(len(raw))

		onReq := func(session *engine.Session, buf []byte) {
			body := string(session.Req.Body.AsBuf(session))
			if body != "hello world" {
				t.Errorf("Expected 'hello world', got %q", body)
			}
		}

		parser.Parse(s, onReq)
	})

	t.Run("Incremental Parsing (Incomplete)", func(t *testing.T) {
		s := &engine.Session{
			Buf: make([]byte, 1024),
		}
		part1 := "GET /index HTTP/1.1\r\nHost: "
		copy(s.Buf, part1)
		s.Offset = uint32(len(part1))

		called := false
		onReq := func(session *engine.Session, buf []byte) { called = true }

		parser.Parse(s, onReq)
		if called {
			t.Error("Callback should not be called for incomplete request")
		}

		part2 := "localhost\r\n\r\n"
		copy(s.Buf[s.Offset:], part2)
		s.Offset += uint32(len(part2))

		_, err := parser.Parse(s, onReq)
		if err != nil || !called {
			t.Error("Should have parsed after getting the rest of data")
		}
	})

	t.Run("Pipelining (Multiple Requests)", func(t *testing.T) {
		s := &engine.Session{
			Buf: make([]byte, 1024),
		}
		req := "GET /1 HTTP/1.1\r\n\r\n"
		raw := req + req
		copy(s.Buf, raw)
		s.Offset = uint32(len(raw))

		count := 0
		onReq := func(session *engine.Session, buf []byte) {
			count++
		}

		parser.Parse(s, onReq)
		if count != 2 {
			t.Errorf("Expected 2 requests to be parsed, got %d", count)
		}
	})
}
