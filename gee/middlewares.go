package gee

import (
	"log"
	"time"
)

func Logger() HandlerFunc {
	return func(ctx *Context) {
		t := time.Now()
		ctx.Next()
		log.Printf("[%d] %s in %v", ctx.StatusCode, ctx.Request.RequestURI, time.Since(t))
	}
}
