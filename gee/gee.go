package gee

import (
	"log"
	"net/http"
)

type HandlerFunc func(ctx *Context)

type RouterGroup struct {
	prefix     string
	middleware []HandlerFunc // 这个暂时用不到
	parent     *RouterGroup  // 如果 engine 全部存储的话，parent 就没什么作用了
	engine     *Engine
}

type Engine struct {
	*RouterGroup
	router *router
	groups []*RouterGroup
}

func (g *RouterGroup) Group(prefix string) *RouterGroup {
	engine := g.engine
	newGroup := &RouterGroup{
		prefix:     g.prefix + prefix,
		middleware: make([]HandlerFunc, 0),
		parent:     g,
		engine:     g.engine,
	}
	engine.groups = append(engine.groups, newGroup)
	return newGroup
}

func (g *RouterGroup) GET(pattern string, f HandlerFunc) {
	pattern = g.prefix + pattern
	g.engine.addRoute("GET", pattern, f)
}

func (g *RouterGroup) POST(pattern string, f HandlerFunc) {
	pattern = g.prefix + pattern
	g.engine.addRoute("POST", pattern, f)
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(w, r)
	e.router.handle(ctx)
}

func (e *Engine) addRoute(method string, pattern string, f HandlerFunc) {
	e.router.addRoute(method, pattern, f)
}

func (e *Engine) Run(addr string) error {
	log.Printf("Gee Start! Listen Request on %v", addr)
	return http.ListenAndServe(addr, e)
}

func New() *Engine {
	engine := &Engine{router: newRouter()}
	engine.RouterGroup = &RouterGroup{engine: engine}
	engine.groups = []*RouterGroup{engine.RouterGroup}
	return engine
}
