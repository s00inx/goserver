run:
	go run ./cmd/main.go

push:
	git add .
	git commit -m "$(m)"
	git push origin main

bmem:
	go test -bench=. -benchmem -memprofile=mem.out
	go tool pprof -http=:8000 mem.out