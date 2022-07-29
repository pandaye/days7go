package main

import (
	"context"
	"geerpc"
	"geerpc/registry"
	"geerpc/xclient"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type Foo int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (f Foo) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Num1))
	*reply = args.Num1 + args.Num2
	return nil
}

func startRegistry(wg *sync.WaitGroup) {
	l, err := net.Listen("tcp", ":9999")
	log.Println("done start registry", l.Addr().String(), "listening...")
	if err != nil {
		panic(err)
	}
	registry.HandleHTTP()
	wg.Done()

	// 这里直接用 http 的原因是 http 自带的 ServerMux handler
	_ = http.Serve(l, nil)
}

func startServer(registryAddr string, wg *sync.WaitGroup) {
	var foo Foo
	server := geerpc.NewServer()
	if err := server.Register(&foo); err != nil {
		log.Fatal("register error:", err)
	}

	// 创建监听 socket
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalln("server start error: ", err)
	}
	log.Println("listen: ", l.Addr().String())

	// XDial 用 proto@addr 格式定义连接
	registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	wg.Done()
	server.Accept(l)
}

func foo(xc *xclient.XClient, ctx context.Context, typ, method string, args *Args) {
	var reply int
	var err error
	switch typ {
	case "call":
		err = xc.Call(ctx, method, args, &reply)
	case "broadcast":
		err = xc.BroadCast(ctx, method, args, &reply)
	default:
		log.Println("Err: not supported call type")
	}
	if err != nil {
		log.Printf("%s %s error: %v", typ, method, err)
	} else {
		log.Printf("%s %s success: %d + %d = %d", typ, method, args.Num1, args.Num2, reply)
	}
}

func call(registryPath string) {
	d := xclient.NewGeeMultiDiscovery(registryPath, 0)
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "call", "Foo.Sum", &Args{i, i})
		}(i)
	}
	wg.Wait()
}

func broadcast(registryPath string) {
	d := xclient.NewGeeMultiDiscovery(registryPath, 0)
	xc := xclient.NewXClient(d, xclient.RandomSelect, nil)
	defer func() { _ = xc.Close() }()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(xc, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			// expect 2 - 5 timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			foo(xc, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func main() {
	log.SetFlags(0)

	registryServer := "http://localhost:9999/_geerpc/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	wg.Add(2)
	go startServer(registryServer, &wg)
	go startServer(registryServer, &wg)
	wg.Wait()

	time.Sleep(time.Second)
	call(registryServer)
	broadcast(registryServer)
	// 当发送间隔过快的话，会出现粘包
}
