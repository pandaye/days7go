package singleflight

import (
	"sync"
)

type call struct {
	val interface{}
	err error
	wg  sync.WaitGroup
}

type Group struct {
	m  sync.Mutex
	cm map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (val interface{}, err error) {
	g.m.Lock()
	if g.cm == nil {
		g.cm = make(map[string]*call)
	}
	if c, ok := g.cm[key]; ok {
		g.m.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.cm[key] = c
	g.m.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.m.Lock()
	delete(g.cm, key)
	g.m.Unlock()

	return c.val, c.err
}
