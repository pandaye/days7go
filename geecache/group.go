package geecache

import (
	"fmt"
	"log"
	"sync"
)

type Group struct {
	name      string
	getter    Getter
	mainCache cache
}

var (
	m      sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(getter Getter, name string, size int64) *Group {
	if getter == nil {
		panic("getter is nil")
	}
	m.Lock()
	defer m.Unlock()
	if _, ok := groups[name]; ok {
		panic("groups exist")
	}
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: newCache(size),
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	m.RLocker()
	defer m.RUnlock()
	if g, ok := groups[name]; ok {
		return g
	}
	return nil
}

func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}

	if bv, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return bv, nil
	}

	return g.load(key)
}

func (g *Group) load(key string) (ByteView, error) {
	return g.loadLocally(key)
}

func (g *Group) loadLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	cached := ByteView{bytes: cloneBytes(bytes)}
	g.mainCache.add(key, cached)
	return cached, nil
}
