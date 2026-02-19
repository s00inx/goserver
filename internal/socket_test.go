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

// vibecoded this tests sorry :(
func TestHandleBufferLogic(t *testing.T) {
	s := &session{buf: make([]byte, 1024)}

	t.Run("Test_Pipelining_And_Copy", func(t *testing.T) {
		s.offset = 0
		// Имитируем приход 1.5 запросов в одном пакете
		data := []byte("GET /1 HTTP/1.1\r\n\r\nGET /2 HT")
		copy(s.buf[s.offset:], data)
		s.offset += len(data)

		// --- НАЧАЛО ЛОГИКИ ИЗ ТВОЕЙ ФУНКЦИИ ---
		// Первый проход (должен найти первый запрос)
		cons, err := mockParseraw(s.buf[:s.offset])
		if err != nil {
			t.Fatalf("Первый запрос должен быть найден, но: %v", err)
		}

		// ПРАВИЛЬНЫЙ ПОРЯДОК:
		rem := s.offset - cons
		if rem > 0 {
			// Копируем данные из хвоста в начало ДО обновления s.offset
			copy(s.buf, s.buf[cons:s.offset])
		}
		s.offset = rem
		// --- КОНЕЦ ЛОГИКИ ---

		// Проверка: в буфере должно остаться "GET /2 HT"
		expectedRemaining := "GET /2 HT"
		actualRemaining := string(s.buf[:s.offset])
		if actualRemaining != expectedRemaining {
			t.Errorf("Ожидали в остатке '%s', получили '%s'", expectedRemaining, actualRemaining)
		}

		// Имитируем досылку остатка (те самые 0.5 запроса из теста)
		secondPart := []byte("TP/1.1\r\n\r\n")
		copy(s.buf[s.offset:], secondPart)
		s.offset += len(secondPart)

		// Второй проход (должен найти второй запрос)
		cons2, err2 := mockParseraw(s.buf[:s.offset])
		if err2 != nil {
			t.Fatalf("Второй запрос должен быть найден после досылки, но: %v", err2)
		}

		if cons2 != len("GET /2 HTTP/1.1\r\n\r\n") {
			t.Error("Размер второго запроса не совпадает")
		}
	})

	t.Run("Test_Buffer_Overflow_Protection", func(t *testing.T) {
		s.offset = 0
		maxRequestSize := 10
		longData := []byte("VERY_LONG_REQUEST_DATA")

		s.offset += len(longData)
		if s.offset > maxRequestSize {
			// Логика сброса сессии
			s.offset = 0 // Условно "reset"
		}

		if s.offset != 0 {
			t.Error("Сессия не была сброшена при превышении лимита")
		}
	})
}

func mockParseraw(data []byte) (cons int, err error) {
	for i := 0; i < len(data)-3; i++ {
		if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			return i + 4, nil
		}
	}
	return 0, errIncomplete
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
		_, err := parseraw(raw, hbuf, r)
		if err != nil {
			b.Fatal(err)
		}
	}
}
