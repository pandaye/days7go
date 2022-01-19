package gee

import (
	"fmt"
	"reflect"
	"testing"
)

func newTestRouter() *router {
	r := newRouter()
	r.addRoute("GET", "/", nil)
	r.addRoute("GET", "/hello/:name", nil)
	r.addRoute("GET", "/hello/b/c", nil)
	r.addRoute("GET", "/hi/:name", nil)
	r.addRoute("GET", "/assets/*filepath", nil)
	return r
}

func displayTrieRoot(method string, v *node) {
	fmt.Println(method)
	m := make([]*node, 1)
	m[0] = v
	head := 0
	for len(m) > 0 {
		m = m[head:]
		head = len(m)
		for i := 0; i < head; i++ {
			fmt.Printf("%s-", m[i].part)
			m = append(m, m[i].children...)
		}
		fmt.Println()
	}
}

func TestParsePattern(t *testing.T) {
	ok := reflect.DeepEqual(parsePattern("/p/:name"), []string{"p", ":name"})
	ok = ok && reflect.DeepEqual(parsePattern("/p/*"), []string{"p", "*"})
	ok = ok && reflect.DeepEqual(parsePattern("/p/*name/*"), []string{"p", "*name"})
	if !ok {
		t.Fatal("test parsePattern failed")
	}
}

func TestAddRoute(t *testing.T) {
	r := newTestRouter()
	if _, ok := r.roots["GET"]; !ok {
		t.Fatal("route add failed")
	}
}

func TestGetRoute(t *testing.T) {
	r := newTestRouter()
	n, ps := r.getRoute("GET", "/assets/css/geektutu.css")

	if n == nil {
		t.Fatal("nil shouldn't be returned")
	}

	if n.pattern != "/assets/*filepath" {
		t.Fatal("should match /hello/:name")
	}

	if ps["filepath"] != "css/geektutu.css" {
		t.Fatal("name should be equal to 'geektutu'")
	}

	fmt.Printf("matched path: %s, params['name']: %s\n", n.pattern, ps["filepath"])

}
