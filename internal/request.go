package internal

import (
	"bytes"
	"errors"
)

type request struct {
	method   []byte
	path     []byte
	protocol []byte

	headers []header
	body    []byte
}

type header struct {
	key, val []byte
}

var (
	availablem = [][]byte{
		[]byte("GET"),
		[]byte("POST"),
		[]byte("PUT"),
		[]byte("PATCH"),
		[]byte("DELETE"),
	}
	errInvalid    = errors.New("invalid request")
	errIncomplete = errors.New("incomplete request")
)

// parse raw bytes to request struct from session w zero-alloc
// input raw data bytes, buffer for headers (for 0 alloc), request ptr from session struct
func parseraw(raw []byte, hbuf []header, req *request) (int, error) {
	*req = request{}
	crs := 0
	req.headers = hbuf[:0]

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
	req.method = raw[crs:sep]

	// check if request method is valid
	isvalid := false
	for _, me := range availablem {
		if bytes.Equal(me, req.method) {
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
	req.path = raw[crs:sep]
	crs = sep + 1

	// find request protocol (basically HTTP\1.1)
	sep = findsep(crs, '\n')
	if sep == -1 {
		return 0, errIncomplete
	}
	if sep > crs && raw[sep-1] == '\r' {
		req.protocol = raw[crs : sep-1]
		crs = sep + 1
	} else {
		return 0, errInvalid
	}

	// find request headers
	var contentlen int
	clh := []byte("Content-Length")
	for {
		if crs+1 >= len(raw) {
			return 0, errIncomplete
		}

		if raw[crs] == '\r' && raw[crs+1] == '\n' {
			crs += 2
			break
		}

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

		if len(req.headers) < cap(hbuf) {
			req.headers = append(req.headers, header{key, val})
		}

		if len(key) == 14 && bytes.EqualFold(clh, key) {
			for _, c := range val {
				if c >= '0' && c <= '9' {
					contentlen = contentlen*10 + int(c-'0')
				}
			}
		}

		crs = lf + 1
	}

	if contentlen > 0 {
		if crs+contentlen > len(raw) {
			return 0, errIncomplete
		}
		req.body = raw[crs : crs+contentlen]
		crs += contentlen
	}

	return crs, nil
}
