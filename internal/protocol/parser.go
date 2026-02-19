// parse raw bytes to HTTP request struct w zero-alloc
// onlu parser logic
package protocol

import (
	"bytes"
	"errors"
	"syscall"

	"github.com/kfcemployee/goserver/internal/engine"
)

var (
	// available request methods
	availablem = [][]byte{
		[]byte("GET"),
		[]byte("POST"),
		[]byte("PUT"),
		[]byte("PATCH"),
		[]byte("DELETE"),
	}
)

// stateless HTTPParser struct
// should be init in server.go
type HTTPParser struct{}

// parse raw bytes to request struct from session w zero-alloc
func (p *HTTPParser) Parse(fd int, s *engine.Session) error {
	var err error
	for {
		cons, parserr := p.parseRaw(s.Buf[:s.Offset], s.Hbuf[:], &s.Req)
		if parserr == nil {
			syscall.Write(fd, []byte("HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nOK"))

			rem := s.Offset - cons
			if rem > 0 {
				copy(s.Buf, s.Buf[cons:s.Offset])
			}
			s.Offset = rem
			s.Req = engine.Request{}

			if s.Offset == 0 {
				break
			}
			continue
		} else if errors.Is(parserr, errIncomplete) {
			break
		} else {
			err = parserr
			break
		}
	}

	if err != nil {
		return err
	}
	return nil
}

// input raw data bytes, buffer for headers, request ptr from session struct
func (p *HTTPParser) parseRaw(raw []byte, hbuf []engine.Header, req *engine.Request) (int, error) {
	crs := 0
	req.Headers = hbuf[:0]

	// find a separator
	findsep := func(start int, sep byte) int {
		idx := bytes.IndexByte(raw[start:], sep)
		if idx == -1 {
			return -1
		}
		return start + idx
	}

	// find request method
	sep := findsep(crs, ' ')
	if sep == -1 {
		return 0, errIncomplete
	}
	req.Method = raw[crs:sep]

	// check if request method is valid
	isvalid := false
	for _, me := range availablem {
		if bytes.Equal(me, req.Method) {
			isvalid = true
			break
		}
	}

	if !isvalid {
		return 0, errInvalid
	}
	crs = sep + 1

	// find request path
	sep = findsep(crs, ' ')
	if sep == -1 {
		return 0, errIncomplete
	}
	req.Path = raw[crs:sep]
	crs = sep + 1

	// find request protocol (basically HTTP\1.1)
	sep = findsep(crs, '\n')
	if sep == -1 {
		return 0, errIncomplete
	}
	if sep > crs && raw[sep-1] == '\r' {
		req.Protocol = raw[crs : sep-1]
		crs = sep + 1
	} else {
		return 0, errInvalid
	}

	// find request headers
	var contentlen int
	clh := []byte("Content-Length")
	for {
		// check if we are out of bounds
		if crs+1 >= len(raw) {
			return 0, errIncomplete
		}

		// CRLF means that headers is over
		if raw[crs] == '\r' && raw[crs+1] == '\n' {
			crs += 2
			break
		}

		// header parsing process
		lf := findsep(crs, '\n')
		if lf == -1 {
			return 0, errIncomplete
		}
		if raw[lf-1] != '\r' {
			return 0, errInvalid
		}

		le := lf - 1
		coloni := findsep(crs, ':')
		if coloni == -1 || coloni > le {
			return 0, errInvalid
		}

		vals := coloni + 1
		for vals < le && raw[vals] == ' ' {
			vals++
		}

		key := raw[crs:coloni]
		val := raw[vals:le]

		// max header count is 64 so we need to check overflow
		if len(req.Headers) < cap(hbuf) {
			req.Headers = append(req.Headers, engine.Header{
				Key: key,
				Val: val})
		}

		// find content-length header for body
		// note: no Content-Lentgth means req has NO body
		if len(key) == 14 && bytes.EqualFold(clh, key) {
			for _, c := range val {
				if c >= '0' && c <= '9' {
					contentlen = contentlen*10 + int(c-'0')
				}
			}
		}

		crs = lf + 1
	}

	// parsing body
	if contentlen > 0 {
		if crs+contentlen > len(raw) {
			return 0, errIncomplete
		}
		req.Body = raw[crs : crs+contentlen]
		crs += contentlen
	}

	return crs, nil
}
