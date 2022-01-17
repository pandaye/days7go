package main

import (
	"fmt"
	"gee"
	"log"
	"net/http"
)

func main() {
	e := gee.New()
	e.GET("/", func(writer http.ResponseWriter, request *http.Request) {
		log.Printf("GET /")
		fmt.Fprintf(writer, "hello gee!")
	})
	e.GET("/header", func(writer http.ResponseWriter, request *http.Request) {
		log.Printf("GET /header")
		for k, v := range request.Header {
			fmt.Fprintf(writer, "header[%q] = %q]\n", k, v)
		}
	})
	addr := ":9999"
	log.Fatal(e.Run(addr))
}
