package geecache

import (
	"fmt"
	"geecache/consistenthash"
	pb "geecache/geecachepb"
	"google.golang.org/protobuf/proto"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type PeerPicker interface {
	PickPeer(key string) (PeerGetter, bool)
}

type PeerGetter interface {
	Get(in *pb.Request, out *pb.Response) error
}

type HttpGetter struct {
	basePath string //
}

func (h *HttpGetter) Get(in *pb.Request, out *pb.Response) error {
	if !strings.HasSuffix(h.basePath, "/") {
		h.basePath += "/"
	}
	u := fmt.Sprintf(
		"%v%v/%v",
		h.basePath,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()))
	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v", resp.Status)
	}

	res, err := ioutil.ReadAll(resp.Body)

	if err = proto.Unmarshal(res, out); err != nil {
		return fmt.Errorf("reading response body: %v", err)
	}

	return nil
}

type HttpPool struct {
	self       string
	basePath   string
	mu         sync.Mutex
	peers      *consistenthash.Map
	httpGetter map[string]*HttpGetter
}

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

func NewHttpPool(addr string) *HttpPool {
	return &HttpPool{
		self:     addr,
		basePath: defaultBasePath,
	}
}

func (s *HttpPool) Log(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (s *HttpPool) Set(peers ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.peers = consistenthash.NewMap(defaultReplicas, nil)
	s.peers.Add(peers...)
	s.httpGetter = make(map[string]*HttpGetter, len(peers))
	for _, peer := range peers {
		s.httpGetter[peer] = &HttpGetter{basePath: peer + defaultBasePath}
	}
}

func (s *HttpPool) PickPeer(key string) (PeerGetter, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if peer := s.peers.Get(key); peer != "" && peer != s.self {
		s.Log("Pick peer %s", peer)
		return s.httpGetter[peer], true
	}
	return nil, false
}

func (s *HttpPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp := r.URL.Path
	if !strings.HasPrefix(rp, s.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	s.Log("%s %s", r.Method, r.URL.Path)

	parts := strings.SplitN(rp[len(s.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	group := GetGroup(parts[0])
	if group == nil {
		http.Error(w, "no such group", http.StatusNotFound)
		return
	}

	data, err := group.Get(parts[1])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := proto.Marshal(&pb.Response{Value: data.ByteSlice()})
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}
