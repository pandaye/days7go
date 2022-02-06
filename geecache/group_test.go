package geecache

import (
	"fmt"
	"log"
	"testing"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGroup_Get(t *testing.T) {
	loadCount := make(map[string]int, len(db))
	gee := NewGroup(GetterFunc(func(key string) ([]byte, error) {
		log.Println("[SlowDB] search key", key)
		if v, ok := db[key]; ok {
			if _, ok := loadCount[key]; !ok {
				loadCount[key] = 0
			}
			loadCount[key]++
			return []byte(v), nil
		}
		return nil, fmt.Errorf("key %s does not exist", key)
	}), "test", 2<<10)

	for k, v := range db {
		// 两次读取，第一次测试 load, 第二次一定命中！
		if bv, err := gee.Get(k); err != nil || bv.String() != v {
			t.Fatal("failed to get value of Tom")
		}
		if _, err := gee.Get(k); err != nil || loadCount[k] > 1 {
			t.Fatalf("cache %s miss", k)
		}
	}

	if view, err := gee.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}
