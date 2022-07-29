package registry

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type GeeRegistry struct {
	servers map[string]*ServerItem
	timeout time.Duration
	mu      sync.Mutex
}

type ServerItem struct {
	Addr  string
	start time.Time
}

const (
	defaultPath    = "/_geerpc/registry"
	defaultTimeout = time.Second * 5
)

func NewRegistry(timeout time.Duration) *GeeRegistry {
	return &GeeRegistry{
		servers: make(map[string]*ServerItem),
		timeout: timeout,
	}
}

var DefaultRegistry = NewRegistry(defaultTimeout)

func (r *GeeRegistry) putServer(addr string) {
	r.mu.Lock()
	if s, ok := r.servers[addr]; ok {
		s.start = time.Now()
	} else {
		r.servers[addr] = &ServerItem{addr, time.Now()}
	}
	r.mu.Unlock()
}

func (r *GeeRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	servers := make([]string, len(r.servers))
	for k, v := range r.servers {
		if r.timeout == 0 || v.start.Add(r.timeout).After(time.Now()) {
			servers = append(servers, v.Addr)
		} else {
			delete(r.servers, k)
		}
	}
	return servers
}

func (r *GeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	// GET 方法返回可用地址
	case "GET":
		w.Header().Set("X-Geerpc-Servers", strings.Join(r.aliveServers(), ","))
	case "POST":
		server := req.Header.Get("X-Geerpc-Servers")
		if server == "" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			r.putServer(server)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *GeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}

func HandleHTTP() {
	DefaultRegistry.HandleHTTP(defaultPath)
}

func Heartbeat(registry, addr string, dur time.Duration) {
	if dur == 0 {
		dur = defaultTimeout - 1*time.Second
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(dur)
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}

func sendHeartbeat(registry, addr string) error {
	// log.Println(addr, "send heart beat to registry", registry)
	h := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Geerpc-Servers", addr)
	if _, err := h.Do(req); err != nil {
		log.Println("rpc server send heartbeat error", err.Error())
		return err
	}
	return nil
}
