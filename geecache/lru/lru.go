package lru

import "container/list"

type Cache struct {
	maxBytes  int64
	nBytes    int64
	ll        *list.List
	cache     map[string]*list.Element
	OnEvicted func(key string, value Value) // 不知道干嘛的
}

type entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

func New(max int64, fun func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  max,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: fun,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		v := ele.Value.(*entry)
		return v.value, true
	}
	return
}

func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() // 这里要判空，因为可能已经删没了？ 嗯？ 怎么会删没？ 缓存是什么
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)
		// 还需要改变当前缓存容量
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		// 调用回调函数
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add 新增或修改？ 为什么是或？
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		v := ele.Value.(*entry)
		c.nBytes += int64(v.value.Len() - value.Len())
		v.value = value
	} else {
		ele = c.ll.PushFront(&entry{key: key, value: value})
		c.cache[key] = ele
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.nBytes > c.maxBytes {
		c.RemoveOldest()
	}
}

func (c *Cache) Len() int {
	return c.ll.Len()
}
