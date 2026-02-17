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
	hc      uint8

	body []byte
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
)

// parse raw bytes to request struct from session w zero-alloc
// input raw data bytes, buffer for headers (for 0 alloc), request ptr from session struct
func parseRequest(raw []byte, hbuf []header, req *request) (*request, error) {
	*req = request{}
	hbuf = hbuf[:0]
	crs := 0

	// find request method
	for i := crs; i < len(raw); i++ {
		if raw[i] == ' ' {
			req.method = raw[crs:i]
			crs = i + 1
			break
		}
	}

	// check if request method is valid
	ise := false
	for _, me := range availablem {
		if bytes.Equal(me, req.method) {
			ise = true
			break
		}
	}
	if !ise {
		return nil, errors.New("invalid request")
	}

	// find request path
	for i := crs; i < len(raw); i++ {
		if raw[i] == ' ' {
			req.path = raw[crs:i]
			crs = i + 1
			break
		}
	}

	// find request protocol (basically HTTP\1.1)
	for i := crs; i < len(raw)-2; i++ {
		if bytes.Equal(raw[i:i+2], []byte("\r\n")) {
			req.protocol = raw[crs:i]
			crs = i + 2
			break
		}
	}

	// find request headers
	for {
		if len(raw[crs:]) >= 2 && raw[crs] == '\r' && raw[crs+1] == '\n' {
			crs += 2
			break
		}

		// find : separator for key and value
		sepi := bytes.IndexByte(raw[crs:], ':')
		if sepi == -1 {
			break
		}

		// find end of line idx is right for val
		endi := bytes.Index(raw[crs:], []byte("\r\n"))
		if endi == -1 {
			break
		}

		// check if there is spaces before
		valstart := sepi + 1
		for valstart < endi && raw[crs+valstart] == ' ' {
			valstart++
		}

		// append header to request
		if req.hc >= 64 {
			break
		}

		hbuf = append(hbuf, header{
			key: raw[crs : crs+sepi],
			val: raw[crs+valstart : crs+endi],
		})

		crs += endi + 2
		req.hc++
	}

	req.body = raw[crs:]
	req.headers = hbuf[:req.hc]

	return req, nil
}
