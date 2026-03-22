run:
	go run ./cmd/main.go

bmem:
	go test -bench=. -benchmem -memprofile=mem.out
	go tool pprof -http=:8000 mem.out

testwrk: 
	wrk -t18 -c1000 -d15s http://127.0.0.1:8080/h

testslow: 
	slowhttptest -c 1000 -H -i 10 -r 200 -t GET -u http://127.0.0.1:8080/h -x 24 -p 3     

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker logs -f goserver