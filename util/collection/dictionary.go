package collection

import (
	"fmt"
	"github.com/zaolab/sunnified/util"
	"sync"
)

type Dict struct {
	util.ValueGetter
	dict  map[interface{}]interface{}
	def   interface{}
	mutex *sync.RWMutex
}

func NewDict(def interface{}, m ...map[interface{}]interface{}) (d *Dict) {
	d = &Dict{
		dict:  make(map[interface{}]interface{}),
		def:   def,
		mutex: &sync.RWMutex{},
	}
	d.ValueGetter = util.ValueGetter(d.getvalue)

	for _, mp := range m {
		d.Update(mp)
	}

	return
}

func (this *Dict) Len() int {
	this.mutex.Lock()
	defer this.mutex.RUnlock()
	return len(this.dict)
}

func (this *Dict) getvalue(key interface{}) (val interface{}, exists bool) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	val, exists = this.dict[key]
	return
}

func (this *Dict) MapValue(key interface{}, value interface{}) (val interface{}) {
	if val = this.Get(key); val != nil && value != nil {
		util.MapValue(value, val)
	}
	return
}

func (this *Dict) Set(key interface{}, value interface{}) Dictionary {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.dict[key] = value
	return this
}

func (this *Dict) SetDefault(key interface{}, value interface{}) Dictionary {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if _, exists := this.dict[key]; !exists {
		this.dict[key] = value
	}
	return this
}

func (this *Dict) Keys() (keys []interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	keys = make([]interface{}, len(this.dict))
	i := 0

	for key := range this.dict {
		keys[i] = key
		i++
	}

	return
}

func (this *Dict) Values() (values []interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	values = make([]interface{}, len(this.dict))
	i := 0

	for _, val := range this.dict {
		values[i] = val
		i++
	}

	return
}

func (this *Dict) KeysValues() (keys []interface{}, values []interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	keys = make([]interface{}, len(this.dict))
	values = make([]interface{}, len(this.dict))
	i := 0

	for key, val := range this.dict {
		keys[i] = key
		values[i] = val
		i++
	}

	return
}

func (this *Dict) Pairs() (pairs [][2]interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	pairs = make([][2]interface{}, len(this.dict))
	i := 0

	for key, val := range this.dict {
		pairs[i] = [2]interface{}{key, val}
		i++
	}

	return
}

func (this *Dict) HasKey(key interface{}) (exists bool) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	_, exists = this.dict[key]
	return
}

func (this *Dict) Contains(values ...interface{}) bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for _, val := range values {
		if !this.hasValue(val) {
			return false
		}
	}

	return true
}

func (this *Dict) hasValue(value interface{}) bool {
	for _, val := range this.dict {
		if val == value {
			return true
		}
	}

	return false
}

func (this *Dict) KeyOf(value interface{}) interface{} {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for key, val := range this.dict {
		if val == value {
			return key
		}
	}

	return nil
}

func (this *Dict) KeysOf(value interface{}) (keys []interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	return this.keysOf(value)
}

func (this *Dict) keysOf(value interface{}) (keys []interface{}) {
	keys = make([]interface{}, 0, 2)

	for key, val := range this.dict {
		if val == value {
			keys = append(keys, key)
		}
	}

	return
}

func (this *Dict) RemoveAt(key interface{}) (value interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	var exists bool
	if value, exists = this.dict[key]; exists {
		delete(this.dict, key)
	}
	return value
}

func (this *Dict) Remove(value interface{}) (keys []interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	keys = this.keysOf(value)
	for _, key := range keys {
		delete(this.dict, key)
	}
	return
}

func (this *Dict) Pop() (key interface{}, value interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if len(this.dict) > 0 {
		for key, value = range this.dict {
			break
		}
		delete(this.dict, key)
	}
	return
}

func (this *Dict) Update(m map[interface{}]interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	for k, v := range m {
		this.dict[k] = v
	}
}

func (this *Dict) UpdateDictionary(d Dictionary) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	var key, value interface{}
	for iter := d.Iterator(); iter.Next(&key, &value); {
		this.dict[key] = value
	}
}

func (this *Dict) UpdateDict(d *Dict) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for k, v := range d.dict {
		this.dict[k] = v
	}
}

func (this *Dict) UpdateOrderedDict(d *OrderedDict) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for k, v := range d.prevchain {
		this.dict[k] = v.next.value
	}
}

func (this *Dict) ToMap() map[interface{}]interface{} {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.toMap()
}

func (this *Dict) toMap() (m map[interface{}]interface{}) {
	m = make(map[interface{}]interface{})
	for k, v := range this.dict {
		m[k] = v
	}
	return
}

func (this *Dict) Clone() *Dict {
	return &Dict{
		dict:  this.ToMap(),
		def:   this.def,
		mutex: &sync.RWMutex{},
	}
}

func (this *Dict) String() string {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return fmt.Sprintf("%v", this.dict)
}

func (this *Dict) lock() {
	this.mutex.Lock()
}

func (this *Dict) unlock() {
	this.mutex.Unlock()
}

func (this *Dict) Clear() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.dict = make(map[interface{}]interface{})
}

func (this *Dict) Transaction(f func(ExtendedDictionary) bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	clone := &Dict{
		dict:  this.toMap(),
		def:   this.def,
		mutex: &sync.RWMutex{},
	}

	if ok := f(clone); ok {
		this.dict = clone.ToMap()
	}
}

func (this *Dict) IsMatch(f func(interface{}) bool) bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for _, v := range this.dict {
		if f(v) {
			return true
		}
	}

	return false
}

func (this *Dict) Match(f func(interface{}) bool) (interface{}, interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for k, v := range this.dict {
		if f(v) {
			return k, v
		}
	}

	return nil, nil
}

func (this *Dict) Foreach(f func(interface{}, interface{}) bool) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for k, v := range this.dict {
		if !f(k, v) {
			break
		}
	}
}

func (this *Dict) Iterator() (di Iterator) {
	di = &DictIterator{
		dict: this.dict,
	}
	return
}

// this is not thread/concurrent safe
type DictIterator struct {
	dict   map[interface{}]interface{}
	kvchan chan [2]interface{}
	curkv  [2]interface{}
	reset  bool
	initd  bool
}

func (this *DictIterator) init() {
	if !this.initd {
		this.initd = true
		this.kvchan = make(chan [2]interface{})

		go func() {
			defer func() {
				recover()
				close(this.kvchan)
			}()

			for k, v := range this.dict {
				if this.reset {
					break
				}
				var keyval = [2]interface{}{k, v}
				this.kvchan <- keyval
			}
		}()
	}
}

func (this *DictIterator) Next(val ...interface{}) bool {
	if !this.initd {
		this.init()
	}

	keyval, ok := <-this.kvchan

	if !ok {
		this.curkv = [2]interface{}{nil, nil}
		this.initd = false
		return false
	}

	this.curkv = keyval

	if lenval := len(val); lenval == 1 {
		util.MapValue(val[0], keyval[1])
	} else if lenval == 2 {
		util.MapValue(val[0], keyval[0])
		util.MapValue(val[1], keyval[1])
	}

	return true
}

func (this *DictIterator) Get() (interface{}, interface{}) {
	return this.curkv[0], this.curkv[1]
}

func (this *DictIterator) Reset() {
	this.reset = true
	_, _ = <-this.kvchan

	for true {
		_, ok := <-this.kvchan
		if !ok {
			break
		}
	}

	this.reset = false
}
