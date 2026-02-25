package router

import (
	"fmt"
	"testing"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
)

func setSessionView(s *engine.Session, method, path string) {
	s.Reset()

	if s.Buf == nil {
		s.Buf = make([]byte, 1024)
	}

	cur := 0
	// Записываем метод
	copy(s.Buf[cur:], method)
	s.Req.Method = engine.View{St: uint16(cur), End: uint16(cur + len(method))}
	cur += len(method)

	// Записываем путь
	copy(s.Buf[cur:], path)
	s.Req.Path = engine.View{St: uint16(cur), End: uint16(cur + len(path))}
}

func dummyHandler(ctx *Context) {}

func TestRouter_HandleAndServe_WithView(t *testing.T) {
	r := NewHTTPRouter()
	r.Get("/api/v1/user/:id", dummyHandler)
	r.Handle("POST", "/login", dummyHandler)

	tests := []struct {
		name   string
		method string
		path   string
		found  bool
	}{
		{"Exact match GET", "GET", "/api/v1/user/123", true},
		{"Exact match POST", "POST", "/login", true},
		{"Method mismatch", "POST", "/api/v1/user/123", false},
		{"Path mismatch", "GET", "/unknown", false},
	}

	s := &engine.Session{Buf: make([]byte, 1024)}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setSessionView(s, tt.method, tt.path)

			handler := r.Serve(s)
			if (handler != nil) != tt.found {
				t.Errorf("%s: expected found=%v, got handler=%v", tt.name, tt.found, handler)
			}
		})
	}
}

func BenchmarkRouter_Serve_View(b *testing.B) {
	r := NewHTTPRouter()
	r.Get("/api/v1/resource/item/details", dummyHandler)

	s := &engine.Session{Buf: make([]byte, 1024)}
	setSessionView(s, "GET", "/api/v1/resource/item/details")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		h := r.Serve(s)
		if h == nil {
			b.Fatal("route not found")
		}
	}
}

func BenchmarkRouter_Params_Extraction(b *testing.B) {
	r := NewHTTPRouter()
	r.Get("/user/:id/profile", dummyHandler)

	s := &engine.Session{Buf: make([]byte, 1024)}
	setSessionView(s, "GET", "/user/999999/profile")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Req.Pcount = 0
		h := r.Serve(s)
		if h == nil {
			b.Fatal("route not found")
		}
		if s.Req.Pcount == 0 {
			b.Fatal("param not extracted")
		}
	}
}

func BenchmarkRouter_HighLoad_Parallel(b *testing.B) {
	r := NewHTTPRouter()

	r.Get("/api/v1/user/:id", dummyHandler)
	r.Get("/static/js/bundle.js", dummyHandler)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		s := &engine.Session{Buf: make([]byte, 512)}

		for pb.Next() {
			setSessionView(s, "GET", "/api/v1/user/999")
			s.Req.Pcount = 0

			h := r.Serve(s)
			if h == nil {
				b.Fatal("miss")
			}
		}
	})
}

func BenchmarkEngine_FullPipeline_Stress(b *testing.B) {
	r := NewHTTPRouter()
	p := &protocol.HTTPParser{}

	r.Get("/api/data", dummyHandler)

	rawInput := []byte(
		"GET /api/data HTTP/1.1\r\nHost: localhost\r\n\r\n" +
			"GET /api/data HTTP/1.1\r\nHost: localhost\r\n\r\n" +
			"GET /api/data HTTP/1.1\r\nHost: localhost\r\n\r\n",
	)

	b.SetBytes(int64(len(rawInput)))
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		s := &engine.Session{Buf: make([]byte, 4096)}

		onReq := func(s *engine.Session, buf []byte) {
			_ = r.Serve(s)
		}

		for pb.Next() {
			copy(s.Buf, rawInput)
			s.Offset = uint32(len(rawInput))

			_, _ = p.Parse(s, onReq)
		}
	})
}

func BenchmarkRouter_Chaos_Fixed(b *testing.B) {
	r := NewHTTPRouter()

	r.Get("/api/v1/resource/:id/update", dummyHandler)

	for i := 0; i < 1000; i++ {
		path := fmt.Sprintf("/api/v1/resource/%d/action", i)
		r.Get(path, dummyHandler)
	}

	targetPath := "/api/v1/resource/999/action"
	method := "GET"
	raw := []byte(method + targetPath)

	s := &engine.Session{
		Buf:  raw,
		Pbuf: [8]engine.Param{},
	}
	s.Req.Method = engine.View{St: 0, End: uint16(len(method))}
	s.Req.Path = engine.View{St: uint16(len(method)), End: uint16(len(raw))}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		s.Req.Pcount = 0

		h := r.Serve(s)

		if h == nil {
			b.Fatalf("not found: %s", targetPath)
		}
	}
}
