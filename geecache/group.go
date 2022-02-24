package geecache

import (
	"fmt"
	"geecache/singleflight"
	"log"
	"sync"
)

type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peer      PeerPicker
	loader    *singleflight.Group
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
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

func GetGroup(name string) *Group {
	m.RLock()
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

func (g *Group) RegisterPeer(peer PeerPicker) {
	if g.peer != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peer = peer
}

func (g *Group) load(key string) (bv ByteView, err error) {
	value, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peer != nil {
			if peer, ok := g.peer.PickPeer(key); ok {
				if value, err := g.getFromPeers(peer, key); err == nil {
					return value, err
				}
			}
		}
		return g.loadLocally(key)
	})
	if err == nil {
		return value.(ByteView), err
	}
	return
}

func (g *Group) getFromPeers(peer PeerGetter, key string) (ByteView, error) {
	data, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{bytes: cloneBytes(data)}, nil
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
