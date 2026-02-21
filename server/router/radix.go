// radix tree for router logic, it is not acessible from upper packages so use an abstraction: Router
package router

import (
	"bytes"

	"github.com/s00inx/goserver/server/engine"
)

// radix tree node
type node struct {
	prefix  []byte
	ch      []node  // children in flat area for data locality to not miss the cache
	handler Handler // our handler func
	isparam bool    // is node prefix param?
}

// insert node to tree that means link path and handler
func (n *node) insert(path []byte, h Handler) {
	// cut first slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// split our url to segments /api/handler -> {api, handler}
	segm := bytes.Split(path, []byte("/"))
	cur := n

	for _, s := range segm {
		// skip empty route (/)
		if len(s) == 0 {
			continue
		}

		// params starting from : (:id, :name)
		isparam, pref := len(s) > 0 && s[0] == ':', s
		if isparam {
			pref = s[1:]
		}

		// find child index in flat child array
		idx := -1
		for i := range cur.ch {
			if bytes.Equal(cur.ch[i].prefix, pref) {
				idx = i
				break
			}
		}

		// if no target -> make new Node
		if idx == -1 {
			target := node{
				prefix:  pref,
				isparam: isparam,
				ch:      make([]node, 0),
			}
			cur.ch = append(cur.ch, target)
			idx = len(cur.ch) - 1
		}
		cur = &cur.ch[idx]
	}
	// set Node handler
	cur.handler = h
}

// check if req path match any route and parse params,
// we use bytes.IndexByte, and bytes.HasPrefix for zero-alloc byte manipulations
func (n *node) match(path []byte, rreq *engine.RawRequest) Handler {
	// cut first slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	cur := n

	// while any prefix in our path
	for len(path) > 0 {
		found := false

		for i := range cur.ch {
			c := &cur.ch[i]

			if len(c.prefix) > 0 && c.prefix[0] != path[0] && !(*c).isparam {
				continue
			}

			if c.isparam {
				end := bytes.IndexByte(path, '/')
				if end == -1 {
					end = len(path)
				}

				if rreq.Pcount < len(rreq.P) {
					rreq.P[rreq.Pcount] = engine.Param{
						Key: c.prefix,
						Val: path[:end],
					}
					rreq.Pcount++
				}

				path = path[end:]
				cur = c
				found = true
				break
			}

			if bytes.HasPrefix(path, c.prefix) {
				rem := path[len(c.prefix):]
				if len(rem) == 0 || rem[0] == '/' {
					path = rem
					cur = c
					found = true
					break
				}
			}
		}
		if !found {
			return nil // 404
		}

		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
	}
	return cur.handler
}
