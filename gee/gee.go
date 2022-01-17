package gee

import (
	"fmt"
	"log"
	"net/http"
)

type HandlerFunc func(http.ResponseWriter, *http.Request)

type Engine struct {
	router map[string]HandlerFunc
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Method + "-" + r.URL.Path
	if handler, ok := e.router[key]; ok {
		handler(w, r)
	} else {
		fmt.Fprintf(w, "404 NOT FOUND!")
	}
}

func (e *Engine) addRouter(method, pattern string, f HandlerFunc) {
	e.router[method+"-"+pattern] = f
}

func (e *Engine) GET(pattern string, f HandlerFunc) {
	e.addRouter("GET", pattern, f)
}

func (e *Engine) POST(pattern string, f HandlerFunc) {
	e.addRouter("POST", pattern, f)
}

func (e *Engine) Run(addr string) error {
	log.Printf("Gee Start! Listen Request on %v", addr)
	return http.ListenAndServe(addr, e)
}

func New() *Engine {
	return &Engine{router: make(map[string]HandlerFunc)}
}
