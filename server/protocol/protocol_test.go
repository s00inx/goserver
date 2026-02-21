package protocol

import (
	"bytes"
	"errors"
	"testing"

	"github.com/s00inx/goserver/server/engine"
)

func TestParser(t *testing.T) {
	p := &HTTPParser{}

	tests := []struct {
		name    string
		raw     []byte
		wantErr error
		check   func(t *testing.T, req *engine.RawRequest, cons int)
	}{
		{
			name:    "simple_get_RawRequest",
			raw:     []byte("GET /index HTTP/1.1\r\nHost: localhost\r\n\r\n"),
			wantErr: nil,
			check: func(t *testing.T, req *engine.RawRequest, cons int) {
				if !bytes.Equal(req.Method, []byte("GET")) {
					t.Errorf("wrong method: %s", req.Method)
				}
				if !bytes.Equal(req.Path, []byte("/index")) {
					t.Errorf("wrong path: %s", req.Path)
				}
			},
		},
		{
			name:    "post_with_body",
			raw:     []byte("POST /upload HTTP/1.1\r\nContent-Length: 4\r\n\r\ntest"),
			wantErr: nil,
			check: func(t *testing.T, req *engine.RawRequest, cons int) {
				if !bytes.Equal(req.Body, []byte("test")) {
					t.Errorf("wrong body: %s", req.Body)
				}
			},
		},
		{
			name:    "incomplete_RawRequest",
			raw:     []byte("GET /partial HTT"),
			wantErr: errIncomplete,
		},
		{
			name:    "invalid_method",
			raw:     []byte("GOLANG /index HTTP/1.1\r\n\r\n"),
			wantErr: errInvalid,
		},
		{
			name:    "pipelining_first_RawRequest",
			raw:     []byte("GET /1 HTTP/1.1\r\n\r\nGET /2 HTTP/1.1\r\n\r\n"),
			wantErr: nil,
			check: func(t *testing.T, req *engine.RawRequest, cons int) {
				expectedLen := len("GET /1 HTTP/1.1\r\n\r\n")
				if cons != expectedLen {
					t.Errorf("expected consumption %d, got %d", expectedLen, cons)
				}
				if !bytes.Equal(req.Path, []byte("/1")) {
					t.Errorf("wrong path for first RawRequest: %s", req.Path)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hbuf := make([]engine.Header, 64)
			req := &engine.RawRequest{}
			cons, err := p.parseRaw(tc.raw, hbuf, req)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}

			if tc.check != nil && err == nil {
				tc.check(t, req, cons)
			}
		})
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
