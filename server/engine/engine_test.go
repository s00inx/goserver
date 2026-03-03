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

func BenchmarkEpollServer(b *testing.B) {
	addr := [4]byte{127, 0, 0, 1}
	port := 8888
	target := "127.0.0.1:8888"

	go func() {
		if err := StartEpoll(addr, port, mockParse); err != nil {
			return
		}
	}()

	time.Sleep(500 * time.Millisecond)

	// Подготавливаем статический запрос
	req := []byte("GET /h HTTP/1.1\r\nHost: localhost\r\nContent-Length: 0\r\n\r\n")

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		res := make([]byte, 1024)
		conn, err := net.Dial("tcp", target)
		if err != nil {
			b.Errorf("Dial error: %v", err)
			return
		}
		defer conn.Close()
		for pb.Next() {
			_, err := conn.Write(req)
			if err != nil {
				return
			}

			// Читаем ответ
			_, err = conn.Read(res)
			if err != nil {
				return
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
