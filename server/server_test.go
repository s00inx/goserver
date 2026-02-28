package server

import (
	"testing"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/router"
)

func BenchmarkServeHTTP(b *testing.B) {
	r := router.NewHTTPRouter()

	handler := func(c *router.Context) {
		c.SendDirect(200, []byte("hello"))
	}

	r.Get("/h", handler)

	s := &engine.Session{
		Buf: []byte("GET /h HTTP/1.1\r\nHost: localhost\r\n\r\n"),
	}

	s.Req.Method = engine.View{St: 0, End: 3}
	s.Req.Path = engine.View{St: 4, End: 6}
	s.Req.Protocol = engine.View{St: 7, End: 15}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		h := r.Serve(s)

		if h != nil {
			c := ctxPool.Get().(*router.Context)

			c.Reset(s, nil)
			handler(c)

			ctxPool.Put(c)
		}
	}
}
