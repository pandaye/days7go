package geecache

import (
	"log"
	"net/http"
	"strings"
)

type HttpPool struct {
	self     string
	basePath string
}

const defaultBasePath = "/_geecache/"

func NewHttpPool(addr string) *HttpPool {
	return &HttpPool{
		self:     addr,
		basePath: defaultBasePath,
	}
}

func (s *HttpPool) Log(format string, v ...interface{}) {
	log.Printf(format, v...)
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

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data.ByteSlice())
}
