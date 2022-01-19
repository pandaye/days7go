package gee

import (
	"net/http"
	"strings"
)

type node struct {
	pattern  string
	part     string
	children []*node
	isWaild  bool
}

func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part {
			return child
		}
	}
	return nil
}

func (n *node) matchChildren(part string) []*node {
	children := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWaild {
			children = append(children, child)
		}
	}
	return children
}

func (n *node) insert(parts []string) *node {
	m := n
	for _, part := range parts {
		tmp := m.matchChild(part)
		if tmp == nil {
			newNode := &node{
				part:     part,
				children: make([]*node, 0),
				isWaild:  part[0] == ':' || part[0] == '*',
			}
			m.children = append(m.children, newNode)
			m = newNode
		} else {
			m = tmp
		}
	}
	m.pattern = "/" + strings.Join(parts, "/")
	if m.pattern == "/" {
	}
	return m
}

func (n *node) search(parts []string) *node {
	// 如果没有插入 / 则 root.pattern 为空，此时查询 / 应返回 nil
	if len(parts) == 0 && n.pattern == "/" {
		return n
	}
	// 广度优先搜索
	m := make([]*node, 1)
	m[0] = n
	head := 0
	for k, p := range parts {
		m = m[head:]
		head = len(m)
		for i := 0; i < head; i++ {
			children := m[i].matchChildren(p)
			// k + 1 == len 代表刚刚好匹配
			if len(children) > 0 && (k+1 == len(parts) || strings.HasPrefix(children[0].part, "*")) {
				return children[0]
			}
			m = append(m, children...)
		}
	}
	return nil
}

type router struct {
	roots   map[string]*node
	handler map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		roots:   make(map[string]*node),
		handler: make(map[string]HandlerFunc),
	}
}

func parsePattern(pattern string) []string {
	parts := strings.Split(pattern, "/")
	if len(parts) == 0 {
		panic("URL path format error!")
	}
	var i = 0
	for _, s := range parts {
		if s != "" {
			parts[i] = s
			i++
			if s[0] == '*' {
				break
			}
		}
	}
	return parts[:i]
}

func (r *router) addRoute(method, pattern string, f HandlerFunc) {
	if _, ok := r.roots[method]; !ok {
		r.roots[method] = &node{
			pattern:  "",
			part:     "/", // 这个其实无所谓
			children: make([]*node, 0),
			isWaild:  false,
		}
	}
	keyNode := r.roots[method].insert(parsePattern(pattern))

	key := method + "-" + keyNode.pattern
	r.handler[key] = f
}

func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}
	originParts := parsePattern(path)
	keyNode := root.search(originParts)
	if keyNode != nil {
		params := make(map[string]string)
		parts := parsePattern(keyNode.pattern)
		for i, v := range parts {
			if v[0] == ':' {
				params[v[1:]] = originParts[i]
			}
			if v[0] == '*' {
				params[v[1:]] = strings.Join(originParts[i:], "/")
				break
			}
		}
		return keyNode, params
	}
	return nil, nil
}

func (r *router) handle(ctx *Context) {
	keyNode, params := r.getRoute(ctx.Method, ctx.Path)
	if keyNode != nil {
		ctx.Params = params
		handler := r.handler[ctx.Method+"-"+keyNode.pattern]
		ctx.handlers = append(ctx.handlers, handler)
	} else {
		ctx.handlers = append(ctx.handlers, func(ctx *Context) {
			ctx.String(http.StatusNotFound, "Page %v Not Found!", ctx.Path)
		})
	}
	ctx.Next()
}
