package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type HashFunc func([]byte) uint32

type Map struct {
	hash     HashFunc
	replicas int
	keys     []int
	hashmap  map[int]string
}

func NewMap(replicas int, fn HashFunc) *Map {
	hmap := &Map{
		replicas: replicas,
		hash:     fn,
		hashmap:  make(map[int]string),
	}
	if fn == nil {
		hmap.hash = crc32.ChecksumIEEE
	}
	return hmap
}

func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			idx := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, idx)
			m.hashmap[idx] = key
		}
	}
	sort.Ints(m.keys)
}

func (m *Map) Get(key string) string {
	if len(key) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	return m.hashmap[m.keys[idx%len(m.keys)]]
}
