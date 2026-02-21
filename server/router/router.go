// HTTPRouter logic and public methods
package router

// TODO: add persing for all methods

import (
	"github.com/s00inx/goserver/server/engine"
)

// indexes of exact method tree ptr from router
const (
	mGet = iota
	mPost
	mPut
	mDelete
	mUnknown
	mcnt // counts of methods
)

// http router: store only array of tree root ptrs
type HTTPRouter struct {
	trees [mcnt]*node
}

// init new http router with array of ptrs to roots of trees for every method,
// so we can store same paths to GET and POST for example
func NewHTTPRouter() *HTTPRouter {
	r := &HTTPRouter{}
	for i := range mcnt {
		r.trees[i] = &node{ch: make([]node, 0)}
	}
	return r
}

// fast parsing method (and unprotected)
func parseMethod(m []byte) int {
	if len(m) == 0 {
		return mUnknown
	}

	// switch on 1st byte
	switch m[0] {
	case 'G': // GET
		if len(m) == 3 {
			return mGet
		}
	case 'P':
		if len(m) == 4 && m[1] == 'O' {
			return mPost
		} // POST
		if len(m) == 3 && m[1] == 'U' {
			return mPut
		} // PUT
	case 'D':
		return mDelete
	}
	return mUnknown
}

// serve: find a handler to path
func (r *HTTPRouter) Serve(rreq *engine.RawRequest) Handler {
	mi := parseMethod(rreq.Method)
	if mi == mUnknown {
		return nil // 404
	}

	return r.trees[mi].match(rreq.Path, rreq) // 404
}

// common func to link file to path ;
// note: there is 2 allocs when we call []byte(string) but since it's one time it doesnt affect runtime performance
func (r *HTTPRouter) Handle(method, path string, h Handler) {
	mi := parseMethod([]byte(method))

	if mi == mUnknown {
		return
	}

	r.trees[mi].insert([]byte(path), h)
}

// easy method to register GET request, a bit of syntactic sugar :)
// this is no overhead bc compiler will likely inline this call into handle
func (r *HTTPRouter) Get(path string, h Handler) {
	r.Handle("GET", path, h)
}
