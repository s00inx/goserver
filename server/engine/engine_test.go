package engine

import (
	"net"
	"os"
	"syscall"
	"testing"
	"time"
)

func mockParse(s *Session) (bool, error) {
	s.Offset = 0
	s.Req = RawRequest{}
	r := []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK")
	syscall.Write(int(s.Fd), r)
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
	b.ReportAllocs()
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
				b.Errorf("Write error: %v", err)
				break
			}
			if _, err := conn.Read(res); err != nil {
				b.Errorf("Read error: %v", err)
				break
			}
		}
	})
}

var mockPayload = func(dst []byte) int {
	body := []byte("Hello, world! This is a zero-alloc engine test.")
	off := 0
	off += copy(dst[off:], []byte("HTTP/1.1 200 OK\r\n"))
	off += copy(dst[off:], []byte("Content-Type: text/plain\r\n\r\n"))
	off += copy(dst[off:], body)
	return off
}

func BenchmarkWriteBuf(b *testing.B) {
	// Открываем /dev/null, чтобы системный вызов Write работал максимально быстро
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	defer devNull.Close()

	s := &Session{Fd: uint32(devNull.Fd())}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_, err := WriteBuf(s, mockPayload)

		if err != nil {
			b.Fatal(err)
		}
	}
}
