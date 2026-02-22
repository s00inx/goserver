package engine

import (
	"net"
	"testing"
	"time"
)

func mockParse(s *Session) (bool, error) {
	s.Offset = 0
	s.Req = RawRequest{}
	// syscall.Write(_, []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"))
	return true, nil
}

func BenchmarkEpollHTTP(b *testing.B) {
	addr := [4]byte{127, 0, 0, 1}
	port := 8888
	target := "127.0.0.1:8888"

	go func() {
		if err := StartEpoll(addr, port, mockParse); err != nil {
			return
		}
	}()

	for i := range 10 {
		conn, err := net.DialTimeout("tcp", target, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			break
		}
		if i == 9 {
			b.Fatalf("Сервер не поднялся на %s", target)
		}
		time.Sleep(100 * time.Millisecond)
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		conn, err := net.Dial("tcp", target)
		if err != nil {
			b.Errorf("Dial error: %v", err)
			return
		}
		defer conn.Close()

		req := []byte("GET / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 0\r\n\r\n")
		res := make([]byte, 1024)

		for pb.Next() {
			if _, err := conn.Write(req); err != nil {
				return
			}
			if _, err := conn.Read(res); err != nil {
				return
			}
		}
	})
}
