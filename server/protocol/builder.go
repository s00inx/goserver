package protocol

import "github.com/s00inx/goserver/server/engine"

// lookup table for status codes
// i use flat list instead of map bc codes is fixed
var statusTable = [505][]byte{
	// 1xx
	100: []byte("100 Continue"),
	101: []byte("101 Switching Protocols"),

	// 2xx
	200: []byte("200 OK"),
	201: []byte("201 Created"),
	202: []byte("202 Accepted"),
	204: []byte("204 No Content"),

	// 3xx
	301: []byte("301 Moved Permanently"),
	302: []byte("302 Found"),
	304: []byte("304 Not Modified"),

	// 4xx
	400: []byte("400 Bad Request"),
	401: []byte("401 Unauthorized"),
	403: []byte("403 Forbidden"),
	404: []byte("404 Not Found"),
	405: []byte("405 Method Not Allowed"),
	408: []byte("408 Request Timeout"),
	413: []byte("413 Payload Too Large"),

	// 5xx
	500: []byte("500 Internal Server Error"),
	501: []byte("501 Not Implemented"),
	502: []byte("502 Bad Gateway"),
	503: []byte("503 Service Unavailable"),
	504: []byte("504 Gateway Timeout"),
}

// for fast access
var (
	proto = []byte("HTTP/1.1 ")
	crlf  = []byte("\r\n")
	colon = []byte(": ")
)

// helper func to copy int to pre-allocated buf with zero-alloc, buf is dst[n:]
// n should be uint bc / 10 (and % 10) for uints is faster (compiler use division by invariant integers), and our len or code > 0
func IntToBuf(buf []byte, n uint) int {
	if n == 0 {
		buf[0] = '0'
		return 1
	}

	var tmp [20]byte
	i := len(tmp)
	for n > 0 {
		i--
		tmp[i] = byte(n%10) + '0'
		n /= 10
	}
	return copy(buf, tmp[i:])
}

func IntToByte(n int) []byte {
	if n == 0 {
		return []byte{0}
	}

	var tmp [20]byte
	i := len(tmp)
	for n > 0 {
		i--
		tmp[i] = byte(n%10) + '0'
		n /= 10
	}

	return tmp[i:]
}

// build response w zero alloc
func BuildResp(code int, headers []engine.Header, body, dst []byte) int {
	if code < 100 || code > 504 {
		code = 500
	}

	st := statusTable[code]
	if st == nil {
		st = []byte("500 Internal Server Error")
	}

	n := copy(dst, proto)
	n += copy(dst[n:], st)
	n += copy(dst[n:], crlf)

	for _, h := range headers {
		n += copy(dst[n:], h.Key)
		n += copy(dst[n:], colon)
		n += copy(dst[n:], h.Val)
		n += copy(dst[n:], crlf)
	}

	n += copy(dst[n:], crlf)
	if len(body) > 0 {
		n += copy(dst[n:], body)
	}

	return n
}
