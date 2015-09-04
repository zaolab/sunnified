package util

import (
	"encoding/json"
	"github.com/bradfitz/gomemcache/memcache"
	//"log"
)

var mc = Memcache{memcache.New("127.0.0.1:11211")}

type Memcache struct {
	*memcache.Client
}

func (this Memcache) Get(key string, m interface{}) (err error) {
	var item *memcache.Item

	if item, err = this.Client.Get(key); err == nil {
		err = json.Unmarshal(item.Value, m)
	}

	return err
}

func (this Memcache) Set(key string, m interface{}, expiry ...int) (err error) {
	var e int32
	var val []byte
	if len(expiry) > 0 {
		e = int32(expiry[0])
	}

	if val, err = json.Marshal(m); err == nil {
		err = this.Client.Set(&memcache.Item{Key: key, Value: val, Expiration: e})
	}

	return
}

func DefaultMemcache() Memcache {
	return mc
}

func NewMemcache(host ...string) Memcache {
	return Memcache{memcache.New(host...)}
}

func Get(key string, m interface{}) (err error) {
	return mc.Get(key, m)
}

func Set(key string, m interface{}, expiry ...int) (err error) {
	return mc.Set(key, m, expiry...)
}
