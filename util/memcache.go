package util

import (
	"encoding/json"

	"github.com/bradfitz/gomemcache/memcache"
)

var mcache = Memcache{nil}

type Memcache struct {
	*memcache.Client
}

func (mc Memcache) Get(key string, m interface{}) (err error) {
	var item *memcache.Item

	if item, err = mc.Client.Get(key); err == nil {
		err = json.Unmarshal(item.Value, m)
	}

	return err
}

func (mc Memcache) Set(key string, m interface{}, expiry ...int) (err error) {
	var e int32
	var val []byte
	if len(expiry) > 0 {
		e = int32(expiry[0])
	}

	if val, err = json.Marshal(m); err == nil {
		err = mc.Client.Set(&memcache.Item{Key: key, Value: val, Expiration: e})
	}

	return
}

func DefaultMemcache() Memcache {
	if mcache.Client == nil {
		mcache = NewMemcache("127.0.0.1:11211")
	}
	return mcache
}

func NewMemcache(host ...string) Memcache {
	return Memcache{memcache.New(host...)}
}

func Get(key string, m interface{}) (err error) {
	return DefaultMemcache().Get(key, m)
}

func Set(key string, m interface{}, expiry ...int) (err error) {
	return DefaultMemcache().Set(key, m, expiry...)
}
