package internal

import (
	"net"
	"testing"
	"time"
)

// vibecoded this benchmark
func BenchmarkEpollHTTP(b *testing.B) {
	addr := [4]byte{127, 0, 0, 1}
	port := 8888
	target := "127.0.0.1:8888"

	// 1. Запускаем сервер
	go func() {
		if err := EpollRecv(addr, port); err != nil {
			// Если ошибка — мы это увидим в логах
			return
		}
	}()

	// 2. Ждем, пока сервер реально начнет слушать порт
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

	// 3. Запускаем нагрузку
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
			// Пишем запрос
			if _, err := conn.Write(req); err != nil {
				return
			}
			// Читаем ответ
			if _, err := conn.Read(res); err != nil {
				return
			}
		}
	})
}
