package xclient

import (
	"context"
	. "geerpc"
	"io"
	"reflect"
	"sync"
)

type XClient struct {
	d       Discovery
	mode    SelectMode
	option  *Option
	clients map[string]*Client
	mu      sync.Mutex
}

func NewXClient(d Discovery, mode SelectMode, option *Option) *XClient {
	return &XClient{
		d:       d,
		mode:    mode,
		option:  option,
		clients: make(map[string]*Client),
	}
}

func (x *XClient) Close() error {
	x.mu.Lock()
	defer x.mu.Unlock()
	for k, v := range x.clients {
		_ = v.Close()
		delete(x.clients, k)
	}
	return nil
}

func (x *XClient) dial(rpcAddr string) (*Client, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	client, ok := x.clients[rpcAddr]
	if ok && !client.IsAvailable() {
		_ = client.Close()
		delete(x.clients, rpcAddr)
		client = nil
	}
	if client == nil {
		var err error
		client, err = XDial(rpcAddr, x.option)
		if err != nil {
			return nil, err
		}
		x.clients[rpcAddr] = client
	}
	return client, nil
}

func (x *XClient) Call(ctx context.Context, method string, argv interface{}, replyv interface{}) error {
	rpcAddr, err := x.d.Get(x.mode)
	if err != nil {
		return err
	}
	return x.call(rpcAddr, ctx, method, argv, replyv)
}

func (x *XClient) call(rpcAddr string, ctx context.Context, method string, argv, replyv interface{}) error {
	client, err := x.dial(rpcAddr)
	if err != nil {
		return err
	}
	return client.Call(ctx, method, argv, replyv)
}

func (x *XClient) BroadCast(ctx context.Context, method string, argv interface{}, replyv interface{}) error {
	rpcAddrs, err := x.d.GetAll()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var e error
	replyDone := replyv == nil
	nCtx, cancel := context.WithCancel(ctx)
	for _, rpcAddr := range rpcAddrs {
		wg.Add(1)
		go func(rpcAddr string) {
			defer wg.Done()
			var cloneReply interface{}
			if replyv != nil {
				cloneReply = reflect.New(reflect.ValueOf(replyv).Elem().Type()).Interface()
			}
			err := x.call(rpcAddr, nCtx, method, argv, cloneReply)
			mu.Lock()
			if err != nil && e == nil {
				e = err
				cancel()
			}
			if err == nil && !replyDone {
				reflect.ValueOf(replyv).Elem().Set(reflect.ValueOf(cloneReply).Elem())
				// replyv = cloneReply
				replyDone = true
			}
			mu.Unlock()
		}(rpcAddr)
	}
	wg.Wait()
	return e
}

var _ io.Closer = (*XClient)(nil)
