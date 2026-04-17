# goserver

**goserver** is a from-scratch HTTP server written in Go, bypassing the standard `net/http` library. 

This started as an educational pet project to dive deeper and understand how high-load servers work under the hood. My goal was to build a custom event loop using Linux syscalls, implement protocol parsing, and do it all with minimal GC pressure (achieving **0 allocations on the hot path**).

### Architecture & Features

I split the project into three independent layers: the network engine (epoll), the protocol parser, and the router.

* **Network Engine (epoll)**
  Instead of the traditional "one goroutine per connection" model, the server uses asynchronous I/O via `epoll`. A worker pool (sized by CPU cores) reads data directly from sockets.
  * Connection lifecycles are managed using `sync.Pool`.
  * Session state is managed via `atomic.Pointer`s, allowing us to drop heavy mutexes and reduce lock contention.
  * Implemented a **custom Timer Wheel** for efficient O(1) dropping of idle (keep-alive) connections.

* **Protocol Parser (HTTP/1.1)**
  * **Zero-copy parsing:** Bytes aren't copied during request reading. The parser stores start/end indices (`View`) for headers, path, and method directly in the raw byte buffer.
  * Supports HTTP Pipelining (multiple requests in one buffer) and incremental parsing (for fragmented requests).
  * Implemented a double-buffered atomic cache for the `Date` header (required in every HTTP response) to avoid formatting time on every single request.

* **Routing**
  * Built on top of a Radix Trie for fast O(k) route matching.
  * Supports dynamic path parameters (e.g., `/api/user/:id`) — extracted without any heap allocations.
  * Includes support for route groups (`Group`) and basic middlewares (like `Recovery` for panic handling).

### Testing & Benchmarks

I paid close attention to performance and edge cases. The code is heavily covered by benchmarks and unit tests using the standard Go tooling.

* Benchmarks with `b.ReportAllocs()` confirm **0 B/op and 0 allocs/op** across the entire pipeline: from socket read to routing and response building.
* Used `b.RunParallel` for stress-testing the router (simulating hundreds of thousands of concurrent trie lookups).
* The parser is tested against edge cases: stitching incomplete requests, reading bodies via `Content-Length`, and handling malformed headers.

**Performance:**
In local testing (Intel Core Ultra 5, 18 threads) using `wrk`, the server handles roughly **~980k RPS** with an average latency of **~1.13 ms**.

> **Note:** This is strictly an educational project. It runs **only on Linux** (due to epoll syscalls). The HTTP/1.1 spec is intentionally not fully implemented (e.g., no Chunked Transfer Encoding) to keep the focus strictly on raw speed and memory mechanics.

### How to run

Since the engine relies on `epoll`, you need Linux (or WSL) to run it.

```bash
make run        # Run the server (main.go)
make bmem       # Run internal memory benchmarks
make testwrk    # Run load tests (requires 'wrk' utility)
```

### Example Usage

The router API is inspired by popular frameworks like Gin/Fiber. Inside the handler, you get a custom `Context` to interact with the request without triggering allocations.

```go
package main

import (
    srv "github.com/s00inx/goserver/server"
)

func main() {
    s := srv.New()

    // Basic route
    s.Get("/ping", func(c *srv.Context) {
        c.SendDirect(200, []byte("pong"))
    })

    // Route groups with parameters
    api := s.Group("/api/v1")
    api.Get("/user/:id", func(c *srv.Context) {
        // Extracting URL param with zero allocations
        id := c.Param(srv.S2Bytes("id")) 
        
        c.SetHeader([]byte("Content-Type"), []byte("text/plain"))
        c.SendWithBody(id)
    })

    // Start server on 127.0.0.1:8080
    s.Run([4]byte{127, 0, 0, 1}, 8080)
}
```