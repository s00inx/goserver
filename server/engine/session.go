package engine

import "sync/atomic"

// request struct, raw because it refers to bytes so we can't use it in user scope, we have Request for it
// all slices are pointers to session Buf for zero-copy
type RawRequest struct {
	Method   View // http method
	Protocol View // proto

	Path     View   // url
	Hcount   uint16 // header count
	Pcount   uint16 // url param count
	RawQuery View   // url query (? ...) raw bc i wouldn't parse it if not needed

	Body View // req body
}

// view for slice
type View struct {
	St  uint16
	End uint16
}

// view as buffer based on Session
func (v *View) AsBuf(s *Session) []byte {
	return s.Buf[v.St:v.End]
}

// header struct based on views
type HeaderView struct {
	Key, Val View
}

// url params w view, so we need static key and val as view
type Param struct {
	Key []byte
	Val View
}

// header for response, maybe i would redo it to views
type Header struct {
	Key, Val []byte
}

// session is an arena for pre-allocated data
// it manages buffers and fd for HTTPRequest, session is atomical instance for 1 socket fd !
// buf, offset for raw data, hbuf and req is pre-allocated buffer for headers and RawRequest struct from pool
type Session struct {
	raw    any
	bufraw any
	tnext  *Session
	tprev  *Session
	Buf    []byte
	Pbuf   [8]Param
	slot   int
	Fd     uint32
	Offset uint32
	Hbuf   [16]HeaderView
	Req    RawRequest

	inWork atomic.Bool
	_      [12]byte
}

// reset session for put it to pool
func (s *Session) Reset() {
	s.Fd = 0
	s.Offset = 0

	s.tnext = nil
	s.tprev = nil
	s.inWork.Store(false)

	s.Req = RawRequest{}
	s.Req.Hcount = 0
	s.Req.Pcount = 0
}
