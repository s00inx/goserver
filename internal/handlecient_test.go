package internal

import (
	"bytes"
	"testing"
)

func TestParseRequest(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name:    "get",
			raw:     "GET /index.html HTTP/1.1\r\nHost: localhost\r\nUser-Agent: Go-Test\r\n\r\n",
			wantErr: false,
		},
		{
			name:    "with body",
			raw:     "POST /submit HTTP/1.1\r\nContent-Type: text/plain\r\nContent-Length: 5\r\n\r\nhello",
			wantErr: false,
		},
		{
			name:    "invalid method",
			raw:     "Ы /sky HTTP/1.1\r\n\r\n",
			wantErr: true,
		},
		{
			name:    "header w spaces",
			raw:     "GET / HTTP/1.1\r\nCustom-Header:    value-with-space\r\n\r\n",
			wantErr: false,
		},
		{
			name:    "empty",
			raw:     "",
			wantErr: true,
		},
	}

	req := &request{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hbuf := make([]header, 0, 4096)
			req, err := parseRequest([]byte(tt.raw), hbuf, req)

			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRequest() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				if tt.name == "get" {
					if !bytes.Equal(req.method, []byte("GET")) {
						t.Errorf("Expected GET, got %s", req.method)
					}
					if len(req.headers) != 2 {
						t.Errorf("Expected 2 headers, got %d", len(req.headers))
					}
				}

				if tt.name == "with body" {
					if !bytes.Equal(req.body, []byte("hello")) {
						t.Errorf("Expected body 'hello', got %s", req.body)
					}
				}
			}
		})
	}
}

var raw = []byte("GET /api/v1/users/profile?id=12345 HTTP/1.1\r\n" +
	"Host: localhost:8080\r\n" +
	"User-Agent: Mozilla/5.0 (X11; Linux x86_64)\r\n" +
	"Accept: application/json\r\n" +
	"Connection: keep-alive\r\n" +
	"\r\n")

func BenchmarkParseRequest(b *testing.B) {
	r := &request{}
	hbuf := make([]header, 0, 4096)

	b.ReportAllocs()
	b.SetBytes(int64(len(raw)))

	b.ResetTimer()
	for b.Loop() {
		_, err := parseRequest(raw, hbuf, r)
		if err != nil {
			b.Fatal(err)
		}
	}
}
