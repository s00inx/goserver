// prefix tree for router logic, it is not acessible from upper packages so use an abstraction: Router
package router

import (
	"bytes"

	"github.com/s00inx/goserver/server/engine"
)

// tree node
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
			prefCopy := make([]byte, len(pref))
			copy(prefCopy, pref)
			target := node{prefix: prefCopy, isparam: isparam,
				ch: make([]node, 0),
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
func (n *node) match(s *engine.Session) Handler {
	return n.find(s, s.Buf[s.Req.Path.St:s.Req.Path.End], s.Req.Path.St)
}

func (n *node) find(s *engine.Session, fp []byte, curo uint16) Handler {
	if len(fp) > 0 && fp[0] == '/' {
		fp = fp[1:]
		curo++
	}

	if len(fp) == 0 {
		return n.handler
	}
	for i := range n.ch {
		c := &n.ch[i]
		if !c.isparam && bytes.HasPrefix(fp, c.prefix) {
			rem := fp[len(c.prefix):]
			if len(rem) == 0 || rem[0] == '/' {
				if h := c.find(s, rem, curo+uint16(len(c.prefix))); h != nil {
					return h
				}
			}
		}
	}

	for i := range n.ch {
		c := &n.ch[i]
		if c.isparam {
			end := bytes.IndexByte(fp, '/')
			if end == -1 {
				end = len(fp)
			}

			pIdx := s.Req.Pcount
			if pIdx < uint16(cap(s.Pbuf)) {
				s.Pbuf[pIdx] = engine.Param{
					Key: c.prefix,
					Val: engine.View{St: curo, End: curo + uint16(end)},
				}
				s.Req.Pcount++
			}

			if h := c.find(s, fp[end:], curo+uint16(end)); h != nil {
				return h
			}

			s.Req.Pcount = pIdx
		}
	}

	return nil
}
