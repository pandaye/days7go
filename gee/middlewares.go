package gee

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

func Logger() HandlerFunc {
	return func(ctx *Context) {
		t := time.Now()
		ctx.Next()
		log.Printf("[%d] %s in %v", ctx.StatusCode, ctx.Request.RequestURI, time.Since(t))
	}
}

func Recovery() HandlerFunc {
	return func(ctx *Context) {
		defer func() {
			if err := recover(); err != nil {
				message := fmt.Sprintf("%s", err)
				log.Printf("%s\n\n", trace(message))
				ctx.Fail(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		ctx.Next()
	}
}

func trace(message string) string {
	var pcs [32]uintptr // 最多显示 32 层调用？
	var str strings.Builder
	str.WriteString(message + "\n TraceBack:")
	n := runtime.Callers(3, pcs[:])
	for _, pc := range pcs[:n] {
		fn := runtime.FuncForPC(pc)
		file, line := fn.FileLine(pc)
		str.WriteString(fmt.Sprintf("\n\t%s: %d", file, line))
	}
	return str.String()
}
