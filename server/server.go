package server

import (
	"fmt"

	"github.com/s00inx/goserver/server/engine"
	"github.com/s00inx/goserver/server/protocol"
	"github.com/s00inx/goserver/server/router"
)

// New()              - Инициализация сервера, пулов и роутера
// Run(addr, port)    - Запуск epoll, воркер-пула и прослушивания порта
// Stop()             - Graceful shutdown: остановка приема новых соединений и очистка ресурсов
// SetConfig(conf)    - Настройка лимитов (max conns, buffer sizes, worker count)

// GET(path, h)       - Регистрация обработчика для GET запроса
// POST(path, h)      - Регистрация обработчика для POST запроса
// PUT(path, h)       - Регистрация обработчика для PUT запроса
// PATCH(path, h)     - Регистрация обработчика для PATCH запроса
// DELETE(path, h)    - Регистрация обработчика для DELETE запроса
// Handle(meth, p, h) - Универсальный метод для регистрации любого HTTP метода
// Group(prefix)      - Создание группы роутов с общим префиксом (например, /api/v1)

// Use(middleware)    - Добавление глобального перехватчика (логирование, Auth, Recovery)
// OnConnect(cb)      - Коллбэк при установке нового TCP соединения
// OnDisconnect(cb)   - Коллбэк при закрытии соединения

// Static(path, dir)  - Раздача статики (картинки, html, css) из локальной папки
// File(path, file)   - Раздача одного конкретного файла по заданному пути

// WriteString(fd, s) - Вспомогательный метод для отправки текстового ответа
// WriteJSON(fd, obj) - Сериализация и отправка JSON
// WriteStatus(fd, c) - Отправка HTTP заголовка с кодом ответа (200, 404, 500)

type Server struct {
	R   *router.HTTPRouter
	prs protocol.HTTPParser
}

func Test() {
	addr, port := [4]byte{127, 0, 0, 1}, 8080
	srv := Server{
		R:   router.NewHTTPRouter(),
		prs: protocol.HTTPParser{},
	}

	handler1 := func() {
		fmt.Println("OK")
	}

	handler2 := func() {
		fmt.Println("Handler 2")
	}

	srv.R.Get("/", handler1)
	srv.R.Get("/h", handler2)

	parseFunc := func(fd int, s *engine.Session) {
		onReq := func(fd int, req *engine.RawRequest, buf []byte) {
			h := srv.R.Serve(req)

			if h != nil {
				h()
			} else {
				fmt.Println("error")
			}
		}

		srv.prs.Parse(fd, s, onReq)
	}

	engine.StartEpoll(addr, port, parseFunc)
}
