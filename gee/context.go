package gee

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type H map[string]interface{}

// Context 为什么字段要导出？
type Context struct {
	Response http.ResponseWriter
	Request  *http.Request

	Method string
	Path   string
	Params map[string]string

	StatusCode int

	handlers []HandlerFunc
	index    int
}

func (c *Context) Next() {
	c.index++
	// middleware 应该要求一定包含 Next（），并且 handler 在最后，所以此处不必循环
	if c.index < len(c.handlers) {
		c.handlers[c.index](c)
	}
}

func (c *Context) Param(key string) string {
	value := c.Params[key]
	return value
}

func (c *Context) Status(code int) {
	c.StatusCode = code
	c.Response.WriteHeader(code)
}

func (c *Context) SetHeader(key string, value string) {
	c.Response.Header().Set(key, value)
}

func (c *Context) PostForm(key string) string {
	return c.Request.FormValue(key)
}
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

func (c *Context) Data(code int, data []byte) {
	c.Status(code)
	c.Response.Write(data)
}

func (c *Context) HTML(code int, html string) {
	c.SetHeader("Content-Type", "text/html")
	c.Status(code)
	c.Response.Write([]byte(html))
}

func (c *Context) String(code int, format string, v ...interface{}) {
	c.SetHeader("Content-Type", "text/plain")
	c.Status(code)
	c.Response.Write([]byte(fmt.Sprintf(format, v...)))
}

func (c *Context) JSON(code int, v interface{}) {
	c.SetHeader("Content-Type", "application/json")
	c.Status(code)
	encoder := json.NewEncoder(c.Response)
	// 值得注意的是，这里的 error 其实并不会起作用，gin 中如果写入 json 发生错误的话会产生恐慌
	if err := encoder.Encode(v); err != nil {
		http.Error(c.Response, err.Error(), 500)
	}
}

func (c *Context) Fail(code int, err string) {
	c.String(code, err)
}

func newContext(resp http.ResponseWriter, req *http.Request) *Context {
	return &Context{
		Response: resp,
		Request:  req,
		Path:     req.URL.Path,
		Method:   req.Method,
		handlers: make([]HandlerFunc, 0),
		index:    -1,
	}
}
