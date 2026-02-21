package router

import (
	"testing"

	"github.com/s00inx/goserver/server/engine"
)

func dummyHandler(w *ResponseWriter, req *Request) {}

func TestParse_method(t *testing.T) {
	tests := []struct {
		name     string
		method   []byte
		expected int
	}{
		{"valid get", []byte("GET"), mGet},
		{"valid post", []byte("POST"), mPost},
		{"valid put", []byte("PUT"), mPut},
		{"valid delete", []byte("DELETE"), mDelete},
		{"empty method", []byte(""), mUnknown},
		{"unknown method patch", []byte("PATCH"), mUnknown},
		{"invalid method p", []byte("P"), mUnknown},
		{"invalid method po", []byte("PO"), mUnknown},
		{"invalid method g", []byte("G"), mUnknown},
		{"garbage", []byte("GARBAGE"), mUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMethod(tt.method)
			if result != tt.expected {
				t.Errorf("parseMethod(%q) = %d; want %d", tt.method, result, tt.expected)
			}
		})
	}
}

func TestHttp_router(t *testing.T) {
	r := NewHTTPRouter()

	if r == nil {
		t.Fatal("nil router")
	}
	for i := 0; i < mcnt; i++ {
		if r.trees[i] == nil {
			t.Errorf("nil tree root at %d", i)
		}
	}
}

func TestRouter_handle_and_serve(t *testing.T) {
	r := NewHTTPRouter()

	r.Handle("POST", "/api/users", dummyHandler)
	r.Handle("PUT", "/api/users", dummyHandler)
	r.Get("/health", dummyHandler)

	tests := []struct {
		name          string
		reqMethod     []byte
		reqPath       []byte
		expectHandler bool
	}{
		{
			name:          "match exact post",
			reqMethod:     []byte("POST"),
			reqPath:       []byte("/api/users"),
			expectHandler: true,
		},
		{
			name:          "match exact get via get()",
			reqMethod:     []byte("GET"),
			reqPath:       []byte("/health"),
			expectHandler: true,
		},
		{
			name:          "method not allowed",
			reqMethod:     []byte("GET"),
			reqPath:       []byte("/api/users"),
			expectHandler: false,
		},
		{
			name:          "not found path",
			reqMethod:     []byte("POST"),
			reqPath:       []byte("/api/unknown"),
			expectHandler: false,
		},
		{
			name:          "unknown http method",
			reqMethod:     []byte("PATCH"),
			reqPath:       []byte("/api/users"),
			expectHandler: false,
		},
		{
			name:          "empty path",
			reqMethod:     []byte("GET"),
			reqPath:       []byte(""),
			expectHandler: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &engine.RawRequest{
				Method: tt.reqMethod,
				Path:   tt.reqPath,
			}

			handler := r.Serve(req)

			hasHandler := handler != nil
			if hasHandler != tt.expectHandler {
				t.Errorf("match = %v; want %v for %s %s", hasHandler, tt.expectHandler, tt.reqMethod, tt.reqPath)
			}
		})
	}
}

func BenchmarkRouter_serve(b *testing.B) {
	r := NewHTTPRouter()
	r.Get("/api/v1/users/profile", dummyHandler)

	req := &engine.RawRequest{
		Method: []byte("GET"),
		Path:   []byte("/api/v1/users/profile"),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		h := r.Serve(req)
		if h == nil {
			b.Fatal("handler not found")
		}
	}
}

func BenchmarkRouter_massive_routes(b *testing.B) {
	r := NewHTTPRouter()

	for i := 0; i < 1000; i++ {
		path := "/api/v1/resource/" + string(rune(i))
		r.Get(path, dummyHandler)
	}

	targetPath := []byte("/api/v1/resource/999")
	req := &engine.RawRequest{
		Method: []byte("GET"),
		Path:   targetPath,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		h := r.Serve(req)
		if h == nil {
			b.Fatal("handler not found among 1000 routes")
		}
	}
}
