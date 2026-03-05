run:
	go run ./cmd/main.go

push:
	git add .
	git commit -m "$(m)"
	git push origin main

bmem:
	go test -bench=. -benchmem -memprofile=mem.out
	go tool pprof -http=:8000 mem.out

testwrk: 
	wrk -t18 -c1000 -d15s http://127.0.0.1:8080/h

testslow: 
	slowhttptest -c 1000 -H -i 10 -r 200 -t GET -u http://127.0.0.1:8080/h -x 24 -p 3     