// parse raw bytes to HTTP RawRequest struct w zero-alloc
// only parser logic
package protocol

import (
	"bytes"
	"errors"

	"github.com/s00inx/goserver/server/engine"
)

// stateless HTTPParser struct
// should be init in server.go
type HTTPParser struct{}

// callback func for handling parsed data,
// so it is called when parser did full request
type HandleParsedFunc func(s *engine.Session, buf []byte)

// parse raw bytes to RawRequest struct from session w zero-alloc
func (p *HTTPParser) Parse(s *engine.Session, onreq HandleParsedFunc) (bool, error) {
	var err error
	for {
		cons, parserr := p.parseRaw(s.Buf[:s.Offset], s.Hbuf[:], &s.Req)
		if parserr == nil {
			onreq(s, s.Buf[:cons])

			rem := int(s.Offset) - cons
			if rem > 0 {
				copy(s.Buf, s.Buf[cons:s.Offset])
			}
			s.Offset = uint32(rem)
			s.Req = engine.RawRequest{}

			if s.Offset == 0 {
				return true, nil
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
		return false, err
	}
	return false, nil
}

// input raw data bytes, buffer for headers, RawRequest ptr from session struct
func (p *HTTPParser) parseRaw(raw []byte, hbuf []engine.HeaderView, req *engine.RawRequest) (int, error) {
	crs := 0

	// find a separator
	findsep := func(start int, sep byte) int {
		idx := bytes.IndexByte(raw[start:], sep)
		if idx == -1 {
			return -1
		}
		return start + idx
	}

	// find RawRequest method
	sep := findsep(crs, ' ')
	if sep == -1 {
		return 0, errIncomplete
	}
	req.Method = engine.View{
		St:  uint16(crs),
		End: uint16(sep),
	}
	crs = sep + 1

	// find RawRequest path
	sep = findsep(crs, ' ')
	if sep == -1 {
		return 0, errIncomplete
	}
	req.Path = engine.View{
		St:  uint16(crs),
		End: uint16(sep),
	}
	crs = sep + 1

	// find RawRequest protocol (basically HTTP\1.1)
	sep = findsep(crs, '\n')
	if sep == -1 {
		return 0, errIncomplete
	}
	if sep > crs && raw[sep-1] == '\r' {
		req.Protocol = engine.View{
			St:  uint16(crs),
			End: uint16(sep - 1),
		}
		crs = sep + 1
	} else {
		return 0, errInvalid
	}

	// find RawRequest headers
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

		key := engine.View{
			St:  uint16(crs),
			End: uint16(coloni),
		}
		val := engine.View{
			St:  uint16(vals),
			End: uint16(le),
		}

		// max header count is .. so we need to check overflow
		hi := int(req.Hcount)
		if hi < cap(hbuf) {
			hbuf[hi] = engine.HeaderView{Key: key, Val: val}
			req.Hcount++
		}

		// find content-length header for body
		// note: no Content-Lentgth means req has NO body
		if coloni-crs == 14 && (raw[crs] == 'C' || raw[crs] == 'c') {
			if bytes.EqualFold(clh, raw[crs:coloni]) {
				for _, c := range raw[vals:le] {
					if c >= '0' && c <= '9' {
						contentlen = contentlen*10 + int(c-'0')
					}
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
		req.Body = engine.View{
			St:  uint16(crs),
			End: uint16(crs + contentlen),
		}
		crs += contentlen
	}

	return crs, nil
}
