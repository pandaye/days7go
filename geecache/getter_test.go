package geecache

import (
	"reflect"
	"testing"
)

func TestGetterFunc_Get(t *testing.T) {
	f := GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	except := []byte("key")
	if v, _ := f("key"); !reflect.DeepEqual(v, except) {
		t.Errorf("callback failed")
	}
}
