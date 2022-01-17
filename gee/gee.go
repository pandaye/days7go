package gee

import (
	"log"
	"net/http"
)

type HandlerFunc func(ctx *Context)

type Engine struct {
	router *router
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := newContext(w, r)
	e.router.handle(ctx)
}

func (e *Engine) GET(pattern string, f HandlerFunc) {
	e.router.addRouter("GET", pattern, f)
}

func (e *Engine) POST(pattern string, f HandlerFunc) {
	e.router.addRouter("POST", pattern, f)
}

func (e *Engine) Run(addr string) error {
	log.Printf("Gee Start! Listen Request on %v", addr)
	return http.ListenAndServe(addr, e)
}

func New() *Engine {
	return &Engine{router: newRouter()}
}
