// HTTPRouter logic and public methods
package router

// TODO: add persing for all methods

import (
	"bytes"

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
	RouteGroup // basically def router is RouteGroup w empty prefix ("/")

	trees [mcnt]*node // static trees for common methods for constant search

	// dynamic trees means trees for non-common methods,
	// user can basically add any method, i can't filter it
	// trash realisation using 2 slices :(( should use map
	dynTrees []*node
	dynNames []dmentry
}

// dynamic route entry for link id and name
type dmentry struct {
	name []byte
	id   int
}

// init new http router with array of ptrs to roots of trees for every method,
// so we can store same paths to GET and POST for example
func NewHTTPRouter() *HTTPRouter {
	r := &HTTPRouter{}
	for i := range mcnt {
		r.trees[i] = &node{ch: make([]node, 0)}
	}

	r.RouteGroup = RouteGroup{
		prefix:      "",
		router:      r,
		middlewares: []Handler{Recovery},
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
func (r *HTTPRouter) Serve(s *engine.Session) []Handler {
	mi := parseMethod(s.Req.Method.AsBuf(s))

	pb := s.Req.Path.AsBuf(s)
	if idx := bytes.IndexByte(pb, '?'); idx != -1 {
		absi := s.Req.Path.St + uint16(idx)

		s.Req.RawQuery = engine.View{
			St:  absi + 1,
			End: s.Req.Path.End,
		}
		s.Req.Path.End = absi
	}

	// fast search on common REST methods
	if mi != mUnknown {
		return r.trees[mi].match(s)
	}

	// fallback search in dynamic trees
	for _, entry := range r.dynNames {
		if bytes.Equal(entry.name, s.Req.Method.AsBuf(s)) {
			return r.dynTrees[entry.id].match(s)
		}
	}

	return nil
}

// common func to link file to path ;
// note: there is 2 allocs when we call []byte(string) but since it's one time it doesnt affect runtime performance
func (r *HTTPRouter) Handle(method, path string, h []Handler) {
	mb := []byte(method)
	mi := parseMethod(mb)

	// if method in static -> insert and exit
	if mi != mUnknown {
		r.trees[mi].insert([]byte(path), h)
		return
	}

	// check if tree for method is exist
	for _, entry := range r.dynNames {
		if bytes.Equal(entry.name, mb) {
			r.dynTrees[entry.id].insert([]byte(path), h)
			return
		}
	}

	// register new dynamic route
	nid := len(r.dynTrees)
	r.dynNames = append(r.dynNames, dmentry{name: mb, id: nid})
	nn := &node{ch: make([]node, 0)}
	r.dynTrees = append(r.dynTrees, nn)
	nn.insert([]byte(path), h)
}

// Group for routes with general middlewares and prefix
type RouteGroup struct {
	prefix      string
	middlewares []Handler
	router      *HTTPRouter
}

// new Route Group with Prefix (NOTE: we alloc when append but it doesn't affect runtime 0 alloc performance)
func (g *RouteGroup) Group(prefix string) *RouteGroup {
	return &RouteGroup{
		prefix:      g.prefix + prefix,
		middlewares: append([]Handler{}, g.middlewares...),
		router:      g.router,
	}
}

// link Middleware and Route Group
func (g *RouteGroup) Use(mw Handler) {
	g.middlewares = append(g.middlewares, mw)
}

// common func to link route group and path
func (g *RouteGroup) Handle(method, path string, h Handler) {
	fp := g.prefix + path

	ch := make([]Handler, len(g.middlewares)+1)
	copy(ch, g.middlewares)

	ch[len(g.middlewares)] = h

	g.router.Handle(method, fp, ch)
}

// a bit of syntactic sugar =))

func (g *RouteGroup) Get(path string, h Handler)    { g.Handle("GET", path, h) }
func (g *RouteGroup) Post(path string, h Handler)   { g.Handle("POST", path, h) }
func (g *RouteGroup) Put(path string, h Handler)    { g.Handle("PUT", path, h) }
func (g *RouteGroup) Delete(path string, h Handler) { g.Handle("DELETE", path, h) }
