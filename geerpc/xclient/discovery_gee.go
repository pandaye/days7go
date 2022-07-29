package xclient

import (
	"net/http"
	"strings"
	"time"
)

type GeeMultiDiscovery struct {
	*MultiDiscovery
	timeout    time.Duration
	lastUpdate time.Time
	registry   string
}

const (
	defaultUpdateTimeout = time.Second
)

func NewGeeMultiDiscovery(registryPath string, timeout time.Duration) *GeeMultiDiscovery {
	if timeout == 0 {
		timeout = defaultUpdateTimeout
	}
	d := &GeeMultiDiscovery{
		MultiDiscovery: NewMultiDiscovery(make([]string, 0)),
		timeout:        timeout,
		registry:       registryPath,
	}
	return d
}

func (m *GeeMultiDiscovery) Update(servers []string) error {
	return nil
}

func (m *GeeMultiDiscovery) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastUpdate.Add(m.timeout).After(time.Now()) {
		return nil
	}
	resp, err := http.Get(m.registry)
	if err != nil {
		return err
	}
	serversLine := resp.Header.Get("X-Geerpc-Servers")
	servers := strings.Split(serversLine, ",")
	m.servers = make([]string, 0)
	for _, v := range servers {
		if strings.TrimSpace(v) != "" {
			m.servers = append(m.servers, v)
		}
	}
	m.lastUpdate = time.Now()
	return nil
}

func (m *GeeMultiDiscovery) Get(mode SelectMode) (string, error) {
	if err := m.Refresh(); err != nil {
		return "", err
	}
	return m.MultiDiscovery.Get(mode)
}

func (m *GeeMultiDiscovery) GetAll() ([]string, error) {
	if err := m.Refresh(); err != nil {
		return nil, err
	}
	return m.MultiDiscovery.GetAll()
}
