package geerpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
)

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber    int        // MagicNumber marks this a geerpc request
	CodecType      codec.Type // client may choose different Codec to encode body
	ConnectTimeout time.Duration
	HandleTimeout  time.Duration
}

var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodecType:      codec.GobType,
	ConnectTimeout: 5 * time.Second,
}

// invalidRequest is a placeholder for response argv when error occurs
var invalidRequest = struct{}{}

type Server struct {
	serviceMap sync.Map
}

type request struct {
	h            *codec.Header // header of request
	argv, replyv reflect.Value // argv and replyv of request
	svc          *service
	mtype        *methodType
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func (s *Server) Register(rcvr interface{}) error {
	svc := newService(rcvr)
	if _, dup := s.serviceMap.LoadOrStore(svc.name, svc); dup {
		return errors.New("rpc: service already defined: " + svc.name)
	}
	return nil
}

func (s *Server) findService(serviceMethod string) (svc *service, mType *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	svci, ok := s.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service " + serviceName)
		return
	}
	svc = svci.(*service)
	mType = svc.method[methodName]
	if mType == nil {
		err = errors.New("rpc server: can't find method " + methodName)
	}
	return
}

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
	go s.serveCodec(f(conn), opt.HandleTimeout)
}

func (s *Server) serveCodec(c codec.Codec, timeout time.Duration) {
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
		go s.handleRequest(c, req, wg, sending, timeout)
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

func (s *Server) handleRequest(c codec.Codec, req *request, wg *sync.WaitGroup, sending *sync.Mutex, timeout time.Duration) {
	defer wg.Done()
	called := make(chan struct{})
	sent := make(chan struct{})
	go func() {
		err := req.svc.call(req.mtype, req.argv, req.replyv)
		close(called)
		if err != nil {
			req.h.Error = err.Error()
			s.sendResponse(c, req.h, invalidRequest, sending)
			close(sent)
			return
		}
		s.sendResponse(c, req.h, req.replyv.Interface(), sending)
		close(sent)
	}()
	if timeout == 0 {
		<-called
		<-sent
	}
	select {
	case <-called:
		<-sent
	case <-time.After(timeout):
		req.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		s.sendResponse(c, req.h, invalidRequest, sending)
	}
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
	req.svc, req.mtype, err = s.findService(h.ServiceMethod)
	if err != nil {
		return nil, err
	}
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	if err = c.ReadBody(argvi); err != nil {
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

func Register(rcvr interface{}) error { return DefaultServer.Register(rcvr) }
