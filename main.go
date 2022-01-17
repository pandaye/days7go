package main

import (
	"gee"
	"log"
	"net/http"
)

func main() {
	e := gee.New()
	e.GET("/", func(ctx *gee.Context) {
		log.Println("GET /")
		ctx.HTML(http.StatusOK, "<h1>Hello Gee!</h1>")
	})
	e.POST("/json", func(ctx *gee.Context) {
		log.Println("POST /json")
		ctx.JSON(http.StatusOK, gee.H{
			"username": ctx.PostForm("username"),
			"password": ctx.PostForm("password"),
		})
	})
	e.GET("/query", func(ctx *gee.Context) {
		log.Println("GET /query")
		ctx.String(http.StatusOK, "hello %s\n", ctx.Query("user"))
	})
	addr := ":9999"
	log.Fatal(e.Run(addr))
}
