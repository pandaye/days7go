package geerpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"log"
	"net"
	"sync"
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
			err = call.Error
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

func (client *Client) Call(method string, args, reply interface{}) error {
	call := <-client.Do(method, args, reply, nil).Done
	return call.Error
}

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

func Dial(network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOption(opts...)
	if err != nil {
		panic(err)
	}
	conn, err := net.Dial(network, address)
	if err != nil {
		panic(err)
	}
	defer func() {
		if client == nil {
			_ = conn.Close()
		}
	}()
	return newClient(conn, opt)
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
