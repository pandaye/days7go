package main

import (
	"fmt"
	"geecache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	geecache.NewGroup(geecache.GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB] search key", key)
		if v, ok := db[key]; ok {
			return []byte(v), nil
		}
		return nil, fmt.Errorf("key: %s not found", key)
	}), "tom", 2<<10)

	addr := "localhost:9999"
	peers := geecache.NewHttpPool(addr)
	http.ListenAndServe(addr, peers)
}
