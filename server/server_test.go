package server

import (
	"testing"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
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

func BenchmarkFullStackReal(b *testing.B) {
	r := router.NewHTTPRouter()
	prs := protocol.HTTPParser{}

	paths := []string{"/a", "/b", "/c", "/d", "/e", "/f", "/g", "/h", "/i", "/j"}
	for _, p := range paths {
		r.Get(p, func(c *router.Context) {
			c.SendDirect(200, []byte("hello"))
		})
	}

	var inputs [][]byte
	for _, p := range paths {
		inputs = append(inputs, []byte("GET "+p+" HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	}

	s := &engine.Session{Buf: make([]byte, 1024)}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		input := inputs[i%len(inputs)]
		s.Buf[0] = 0

		s.Offset = uint32(copy(s.Buf, input))
		s.Req = engine.RawRequest{}
		_, _ = prs.Parse(s, func(sess *engine.Session, buf []byte) {
			h := r.Serve(sess)
			if h != nil {
				c := ctxPool.Get().(*router.Context)
				c.Reset(sess, h)

				h[0](c)
				ctxPool.Put(c)
			}
		})

	}
}

func BenchmarkTimerWheelUpdate(b *testing.B) {
	tw := engine.NewWheel(15)

	numSessions := 1000
	sessions := make([]*engine.Session, numSessions)
	for i := 0; i < numSessions; i++ {
		sessions[i] = &engine.Session{Fd: uint32(i)}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Имитируем обновление таймера для сессии
		// Как будто пришел запрос
		s := sessions[i%numSessions]
		tw.Update(s)
	}
}

func BenchmarkTimerWheelParallel(b *testing.B) {
	tw := engine.NewWheel(15)
	b.RunParallel(func(pb *testing.PB) {
		s := &engine.Session{Fd: 1} // Имитация сессии на поток
		for pb.Next() {
			tw.Update(s)
		}
	})
}
