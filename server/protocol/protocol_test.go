package protocol

import (
	"bytes"
	"errors"
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
		_ = BuildResp(200, []byte{}, body, dst)
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

	hbuf := make([]engine.Header, 64)
	req := &engine.RawRequest{}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = p.parseRaw(raw, hbuf, req)
		req.Headers = hbuf[:0]
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
		s.Offset = len(raw)
		copy(s.Buf[:], raw)
		s.Req = engine.RawRequest{}

		_, err := parser.Parse(s, func(sess *engine.Session, buf []byte) {
		})

		if err != nil {
			b.Fatal(err)
		}
	}
}

func Test_parser_all_cases(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		expectError  error
		expectCalls  int
		checkRequest func(t *testing.T, req engine.RawRequest)
	}{
		{
			name:        "valid get request",
			raw:         "GET /index.html HTTP/1.1\r\nHost: localhost\r\nUser-Agent: test\r\n\r\n",
			expectError: nil,
			expectCalls: 1,
			checkRequest: func(t *testing.T, req engine.RawRequest) {
				if !bytes.Equal(req.Method, []byte("GET")) {
					t.Error("wrong method")
				}
				if !bytes.Equal(req.Path, []byte("/index.html")) {
					t.Error("wrong path")
				}
				if len(req.Headers) != 2 {
					t.Errorf("expected 2 headers, got %d", len(req.Headers))
				}
			},
		},
		{
			name:        "valid post with body",
			raw:         "POST /api/v1 HTTP/1.1\r\nContent-Length: 11\r\n\r\nhello world",
			expectError: nil,
			expectCalls: 1,
			checkRequest: func(t *testing.T, req engine.RawRequest) {
				if !bytes.Equal(req.Body, []byte("hello world")) {
					t.Error("wrong body")
				}
			},
		},
		{
			name:        "pipelined requests",
			raw:         "GET /1 HTTP/1.1\r\n\r\nGET /2 HTTP/1.1\r\n\r\n",
			expectError: nil,
			expectCalls: 2,
			checkRequest: func(t *testing.T, req engine.RawRequest) {
				if !bytes.Equal(req.Method, []byte("GET")) {
					t.Error("wrong method")
				}
			},
		},
		{
			name:        "incomplete request",
			raw:         "GET /partial HTTP/1.1\r\nHost: local", // No double CRLF
			expectError: nil,
			expectCalls: 0,
		},
		{
			name:        "invalid method",
			raw:         "777 /sky HTTP/1.1\r\n\r\n",
			expectError: errInvalid,
			expectCalls: 0,
		},
		{
			name:        "malformed header",
			raw:         "GET / HTTP/1.1\r\nNoColonHeader\r\n\r\n",
			expectError: errInvalid,
			expectCalls: 0,
		},
		{
			name:        "body incomplete",
			raw:         "POST / HTTP/1.1\r\nContent-Length: 100\r\n\r\nsmall body",
			expectError: nil,
			expectCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &HTTPParser{}

			s := &engine.Session{
				Offset: len(tt.raw),
				Buf:    make([]byte, 4096),
			}
			copy(s.Buf[:], tt.raw)

			calls := 0
			_, err := parser.Parse(s, func(sess *engine.Session, buf []byte) {
				calls++
				if tt.checkRequest != nil {
					tt.checkRequest(t, sess.Req)
				}
			})

			if tt.expectError != nil {
				if !errors.Is(err, tt.expectError) {
					t.Errorf("expected error %v, got %v", tt.expectError, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if calls != tt.expectCalls {
				t.Errorf("expected %d calls, got %d", tt.expectCalls, calls)
			}
		})
	}
}
