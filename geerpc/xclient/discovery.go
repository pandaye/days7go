package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type SelectMode int

const (
	RandomSelect = iota
	RoundRobinSelect
)

type Discovery interface {
	Update(servers []string) error
	Refresh() error
	Get(mode SelectMode) (string, error)
	GetAll() ([]string, error)
}

type MultiDiscovery struct {
	r       *rand.Rand
	mu      sync.RWMutex
	servers []string
	index   int
}

func (m *MultiDiscovery) Update(servers []string) error {
	return nil
}

func (m *MultiDiscovery) Refresh() error {
	return nil
}

func (m *MultiDiscovery) Get(mode SelectMode) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := len(m.servers)
	if n == 0 {
		return "", errors.New("no available server")
	}
	switch mode {
	case RandomSelect:
		return m.servers[m.r.Intn(n)], nil
	case RoundRobinSelect:
		srv := m.servers[m.index%n]
		m.index = (m.index + 1) % n
		return srv, nil
	default:
		return "", errors.New("not supported mode")
	}
}

func (m *MultiDiscovery) GetAll() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	srvs := make([]string, len(m.servers), len(m.servers))
	copy(srvs, m.servers)
	return srvs, nil
}

func NewMultiDiscovery(servers []string) *MultiDiscovery {
	md := &MultiDiscovery{
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
		servers: servers,
	}
	md.index = md.r.Intn(math.MaxInt32 - 1)
	return md
}

var _ Discovery = (*MultiDiscovery)(nil)
