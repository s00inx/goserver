package protocol

var (
	proto = []byte("HTTP/1.1 ")
	crlf  = []byte("\r\n")
)

// helper func to copy int to pre-allocated buf with zero-alloc
func intToBuf(buf []byte, n int) int {
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

// build response from statuscode, body and copy it to dst buffer
func BuildResp(code int, body []byte, dst []byte) int {
	n := copy(dst, proto) // "HTTP/1.1 "

	n += intToBuf(dst[n:], code)
	n += copy(dst[n:], []byte(" OK"))
	n += copy(dst[n:], crlf)

	n += copy(dst[n:], []byte("Content-Length: "))
	n += intToBuf(dst[n:], len(body))
	n += copy(dst[n:], crlf)
	n += copy(dst[n:], crlf)
	n += copy(dst[n:], body)

	return n
}
