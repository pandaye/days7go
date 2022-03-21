package main

import (
	"encoding/json"
	"fmt"
	"geerpc"
	"geerpc/codec"
	"log"
	"net"
	"time"
)

func startServer(addr chan string) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalln("server start error: ", err)
	}
	log.Println("listen: ", l.Addr().String())
	addr <- l.Addr().String()
	geerpc.Accept(l)
}

func main() {
	addr := make(chan string)
	go startServer(addr)

	conn, err := net.Dial("tcp", <-addr)
	defer func() { _ = conn.Close() }()
	if err != nil {
		log.Fatalln("Connect to server fatal: ", err)
	}

	time.Sleep(time.Second)
	option := geerpc.DefaultOption

	err = json.NewEncoder(conn).Encode(option)
	if err != nil {
		log.Fatalln("encode error")
	}

	cc := codec.NewGobCodec(conn)
	for i := 0; i < 5; i++ {
		req := &codec.Header{
			ServiceMethod: "Foo.Sum",
			Seq:           uint64(i),
		}
		err = cc.Write(req, fmt.Sprintf("geerpc req %d", req.Seq))
		if err != nil {
			log.Fatalln("Send Request Error! ")
		}
		_ = cc.ReadHeader(req)
		var replyv string
		_ = cc.ReadBody(&replyv)
		log.Println("reply:", replyv)
	}
}
