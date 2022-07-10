package geerpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	cc       codec.Codec
	header   codec.Header
	mu       sync.Mutex
	sending  sync.Mutex
	pending  map[uint64]*Call
	seq      uint64
	closed   bool
	shutdown bool
}

var ErrShutdown = errors.New("connection is shut down")

func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closed {
		return ErrShutdown
	}
	client.shutdown = true
	return client.cc.Close()
}

func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closed
}

func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	client.header.ServiceMethod = call.Method
	client.header.Seq = seq
	client.header.Error = ""

	if err = client.cc.Write(&client.header, call.Argv); err != nil {
		call := client.removeCall(seq)
		if call != nil {
			call.Error = err
		}
		call.done()
	}
}

func (client *Client) receive() {
	var err error

	for err == nil {
		var header codec.Header
		err = client.cc.ReadHeader(&header)
		if err != nil {
			break
		}
		call := client.removeCall(header.Seq)
		switch {
		case call == nil:
			err = client.cc.ReadBody(nil)
		case header.Error != "":
			call.Error = fmt.Errorf(header.Error)
			err = call.Error
			call.done()
		default:
			err = client.cc.ReadBody(call.Replyv)
			if err != nil {
				call.Error = errors.New("read body error: " + err.Error())
			}
			call.done()
		}
	}
	client.terminateCalls(err)
}

func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closed || client.shutdown {
		return 0, ErrShutdown
	}
	call.Seq = client.seq
	client.seq++
	client.pending[call.Seq] = call
	return call.Seq, nil
}

func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

func (client *Client) terminateCalls(err error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.sending.Lock()
	defer client.sending.Unlock()
	for _, v := range client.pending {
		v.Error = err
		v.done()
	}
}

func (client *Client) Do(method string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		Method: method,
		Argv:   args,
		Replyv: reply,
		Done:   done,
	}
	client.send(call)
	return call
}

func (client *Client) Call(ctx context.Context, method string, args, reply interface{}) error {
	call := client.Do(method, args, reply, nil)
	select {
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}
}

type newClientFunc func(conn net.Conn, opt *Option) (*Client, error)

func newClient(conn net.Conn, opt *Option) (*Client, error) {
	// 发送 option
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("error while sending option json")
		return nil, err
	}

	f := codec.NewCodecFuncMap[opt.CodecType]
	client := &Client{
		cc:      f(conn),
		seq:     uint64(1),
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client, nil
}

func newHttpClient(conn net.Conn, opt *Option) (*Client, error) {
	// 手写 http 协议头？！
	_, _ = io.WriteString(conn, fmt.Sprintf("CONNECT %s HTTP/1.0\n\n", defaultRPCPath))
	// 处理响应
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == connected {
		fmt.Println("connected")
		return newClient(conn, opt)
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	return nil, err
}

type clientResult struct {
	client *Client
	err    error
}

func dialTimeout(newClient newClientFunc, network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOption(opts...)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout(network, address, opt.ConnectTimeout)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	ch := make(chan clientResult)
	go func() {
		client, err := newClient(conn, opt)
		ch <- clientResult{client, err}
	}()
	// 允许无限等待，直接阻塞接受 ch
	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}
	select {
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	case result := <-ch:
		return result.client, result.err
	}
}

func Dial(network, address string, opts ...*Option) (client *Client, err error) {
	return dialTimeout(newClient, network, address, opts...)
}

func DialHTTP(network, address string, opts ...*Option) (*Client, error) {
	return dialTimeout(newHttpClient, network, address, opts...)
}

func XDial(rpcAddr string, opts ...*Option) (*Client, error) {
	parts := strings.Split(rpcAddr, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("rpc client err: wrong format '%s', expect protocol@addr", rpcAddr)
	}
	protocol, addr := parts[0], parts[1]
	switch protocol {
	case "http":
		return DialHTTP("tcp", addr, opts...)
	default:
		return Dial(protocol, addr, opts...)
	}
}

func parseOption(opts ...*Option) (*Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of options is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = DefaultOption.MagicNumber
	if opt.CodecType == "" {
		opt.CodecType = DefaultOption.CodecType
	}
	return opt, nil
}
