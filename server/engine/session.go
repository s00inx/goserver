package engine

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
	bufraw any // 16 + 16 = 32
	Buf    []byte
	// ^-- big session buf ; all Views refer to THIS ;
	// 24 ; buf sets off only when session need it, see workerEpoll func

	Hbuf [16]HeaderView // headers view 128
	Pbuf [8]Param       // params Key and Val view 256

	Req    RawRequest // 24 request :)
	Fd     uint32
	Offset uint32 // 4 + 4 = 8
}

// reset session for put it to pool
func (s *Session) Reset() {
	s.Fd = 0
	s.Offset = 0

	s.Req = RawRequest{}
	s.Req.Hcount = 0
	s.Req.Pcount = 0
}
