package router

// import (
// 	"testing"

// 	"github.com/s00inx/goserver/server/engine"
// )

// func TestRadixRouter(t *testing.T) {
// 	root := node{
// 		ch: make([]node, 0),
// 	}

// 	// Заглушки хендлеров
// 	h1 := func() {}
// 	h2 := func() {}
// 	h3 := func() { t.Log("Param handler called") }

// 	root.insert([]byte("/api/v1/user"), h1)
// 	root.insert([]byte("/api/v1/order"), h2)
// 	root.insert([]byte("/api/v1/user/:id"), h3)

// 	tests := []struct {
// 		name       string
// 		path       string
// 		wantHandle bool
// 		wantParams map[string]string
// 	}{
// 		{"Static Match", "/api/v1/user", true, nil},
// 		{"Static Match Order", "/api/v1/order", true, nil},
// 		{"Param Match", "/api/v1/user/123", true, map[string]string{"id": "123"}},
// 		{"No Match", "/api/v1/unknown", false, nil},
// 		{"Partial Match", "/api/v1", false, nil},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			req := &engine.RawRequest{
// 				P: make([]engine.Param, 5),
// 			}

// 			handler := root.match([]byte(tt.path), req)

// 			if (handler != nil) != tt.wantHandle {
// 				t.Errorf("Match() gotHandler = %v, want %v", handler != nil, tt.wantHandle)
// 			}

// 			if tt.wantParams != nil {
// 				for i := 0; i < req.Pcount; i++ {
// 					p := req.P[i]
// 					if val, ok := tt.wantParams[string(p.Key)]; ok {
// 						if val != string(p.Val) {
// 							t.Errorf("Param %s: got %s, want %s", p.Key, p.Val, val)
// 						}
// 					}
// 				}
// 			}
// 		})
// 	}
// }

// func BenchmarkRouterMatchStatic(b *testing.B) {
// 	root := node{
// 		ch: make([]node, 0),
// 	}
// 	h := func() {}
// 	root.insert([]byte("/api/v1/user/profile/settings"), h)
// 	path := []byte("/api/v1/user/profile/settings")
// 	req := &engine.RawRequest{P: make([]engine.Param, 5)}

// 	b.ResetTimer()
// 	for b.Loop() {
// 		root.match(path, req)
// 	}
// }

// func BenchmarkRouterMatchParam(b *testing.B) {
// 	root := Node{
// 		ch: make([]Node, 0),
// 	}
// 	h := func() {}
// 	root.Insert([]byte("/api/v1/user/:id/posts/:post_id"), h)
// 	path := []byte("/api/v1/user/123/posts/456")
// 	req := &engine.RawRequest{P: make([]engine.Param, 5)}

// 	b.ResetTimer()
// 	for b.Loop() {
// 		req.Pcount = 0
// 		root.Match(path, req)
// 	}
// }
