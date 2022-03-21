package geerpc

import (
	"encoding/json"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber int        // MagicNumber marks this's a geerpc request
	CodecType   codec.Type // client may choose different Codec to encode body
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

type Server struct{}

type request struct {
	h            *codec.Header // header of request
	argv, replyv reflect.Value // argv and replyv of request
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (s *Server) Accept(lis net.Listener) {
	log.Println("Accepting...")
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		log.Println("Get connected")
		go s.ServeConn(conn)
	}
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("Get Option error")
		return
	}
	if opt.MagicNumber != DefaultOption.MagicNumber {
		log.Println("Not Option")
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]
	if f == nil {
		log.Println("No Codec Found")
		return
	}
	go s.serveCodec(f(conn))
}

func (s *Server) serveCodec(c codec.Codec) {
	sending := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	for {
		req, err := s.readRequest(c)
		if err != nil {
			if req == nil {
				break
			}
			log.Println("rpc server: accept error:", err)
			s.sendResponse(c, req.h, invalidRequest, sending) // 是否需要在头中写入 Error
			break
		}
		wg.Add(1)
		go s.handleRequest(c, req, wg, sending)
	}
	wg.Wait()
	return
}

func (s *Server) sendResponse(c codec.Codec, header *codec.Header, resp interface{}, mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()

	err := c.Write(header, resp)
	if err != nil {
		panic(err) //
	}
}

func (s *Server) handleRequest(c codec.Codec, req *request, wg *sync.WaitGroup, sending *sync.Mutex) {
	defer wg.Done()

	req.replyv = req.argv
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
	s.sendResponse(c, req.h, req.replyv.Interface(), sending)
}

func (s *Server) readRequest(c codec.Codec) (req *request, err error) {
	var h codec.Header
	err = c.ReadHeader(&h)
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	req = &request{h: &h}
	req.argv = reflect.New(reflect.TypeOf("")) // new(string)， 后面用 v 可以吗？
	if err = c.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

func (s *Server) readRequestHeader(c codec.Codec) (h *codec.Header, err error) {
	err = c.ReadHeader(h)
	return h, nil
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}
