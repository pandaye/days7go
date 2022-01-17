package gee

import "net/http"

type router struct {
	handler map[string]HandlerFunc
}

func newRouter() *router {
	return &router{
		handler: make(map[string]HandlerFunc),
	}
}

func (r *router) addRouter(method, pattern string, f HandlerFunc) {
	r.handler[method+"-"+pattern] = f
}

func (r *router) handle(ctx *Context) {
	key := ctx.Method + "-" + ctx.Path
	if handler, ok := r.handler[key]; ok {
		handler(ctx)
	} else {
		ctx.String(http.StatusNotFound, "Page %v Not Found!", ctx.Path)
	}
}
