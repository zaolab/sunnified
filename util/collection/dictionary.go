package collection

import (
	"fmt"
	"sync"

	"github.com/zaolab/sunnified/util"
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

func (d *Dict) Len() int {
	d.mutex.Lock()
	defer d.mutex.RUnlock()
	return len(d.dict)
}

func (d *Dict) getvalue(key interface{}) (val interface{}, exists bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	val, exists = d.dict[key]
	return
}

func (d *Dict) MapValue(key interface{}, value interface{}) (val interface{}) {
	if val = d.Get(key); val != nil && value != nil {
		util.MapValue(value, val)
	}
	return
}

func (d *Dict) Set(key interface{}, value interface{}) Dictionary {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.dict[key] = value
	return d
}

func (d *Dict) SetDefault(key interface{}, value interface{}) Dictionary {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if _, exists := d.dict[key]; !exists {
		d.dict[key] = value
	}
	return d
}

func (d *Dict) Keys() (keys []interface{}) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	keys = make([]interface{}, len(d.dict))
	i := 0

	for key := range d.dict {
		keys[i] = key
		i++
	}

	return
}

func (d *Dict) Values() (values []interface{}) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	values = make([]interface{}, len(d.dict))
	i := 0

	for _, val := range d.dict {
		values[i] = val
		i++
	}

	return
}

func (d *Dict) KeysValues() (keys []interface{}, values []interface{}) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	keys = make([]interface{}, len(d.dict))
	values = make([]interface{}, len(d.dict))
	i := 0

	for key, val := range d.dict {
		keys[i] = key
		values[i] = val
		i++
	}

	return
}

func (d *Dict) Pairs() (pairs [][2]interface{}) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	pairs = make([][2]interface{}, len(d.dict))
	i := 0

	for key, val := range d.dict {
		pairs[i] = [2]interface{}{key, val}
		i++
	}

	return
}

func (d *Dict) HasKey(key interface{}) (exists bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	_, exists = d.dict[key]
	return
}

func (d *Dict) Contains(values ...interface{}) bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for _, val := range values {
		if !d.hasValue(val) {
			return false
		}
	}

	return true
}

func (d *Dict) hasValue(value interface{}) bool {
	for _, val := range d.dict {
		if val == value {
			return true
		}
	}

	return false
}

func (d *Dict) KeyOf(value interface{}) interface{} {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for key, val := range d.dict {
		if val == value {
			return key
		}
	}

	return nil
}

func (d *Dict) KeysOf(value interface{}) (keys []interface{}) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.keysOf(value)
}

func (d *Dict) keysOf(value interface{}) (keys []interface{}) {
	keys = make([]interface{}, 0, 2)

	for key, val := range d.dict {
		if val == value {
			keys = append(keys, key)
		}
	}

	return
}

func (d *Dict) RemoveAt(key interface{}) (value interface{}) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	var exists bool
	if value, exists = d.dict[key]; exists {
		delete(d.dict, key)
	}
	return value
}

func (d *Dict) Remove(value interface{}) (keys []interface{}) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	keys = d.keysOf(value)
	for _, key := range keys {
		delete(d.dict, key)
	}
	return
}

func (d *Dict) Pop() (key interface{}, value interface{}) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if len(d.dict) > 0 {
		for key, value = range d.dict {
			break
		}
		delete(d.dict, key)
	}
	return
}

func (d *Dict) Update(m map[interface{}]interface{}) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	for k, v := range m {
		d.dict[k] = v
	}
}

func (d *Dict) UpdateDictionary(dt Dictionary) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	var key, value interface{}
	for iter := dt.Iterator(); iter.Next(&key, &value); {
		d.dict[key] = value
	}
}

func (d *Dict) UpdateDict(dt *Dict) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	dt.mutex.RLock()
	defer dt.mutex.RUnlock()

	for k, v := range dt.dict {
		d.dict[k] = v
	}
}

func (d *Dict) UpdateOrderedDict(od *OrderedDict) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	for k, v := range od.prevchain {
		d.dict[k] = v.next.value
	}
}

func (d *Dict) ToMap() map[interface{}]interface{} {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.toMap()
}

func (d *Dict) toMap() (m map[interface{}]interface{}) {
	m = make(map[interface{}]interface{})
	for k, v := range d.dict {
		m[k] = v
	}
	return
}

func (d *Dict) Clone() *Dict {
	return &Dict{
		dict:  d.ToMap(),
		def:   d.def,
		mutex: &sync.RWMutex{},
	}
}

func (d *Dict) String() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return fmt.Sprintf("%v", d.dict)
}

func (d *Dict) lock() {
	d.mutex.Lock()
}

func (d *Dict) unlock() {
	d.mutex.Unlock()
}

func (d *Dict) Clear() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.dict = make(map[interface{}]interface{})
}

func (d *Dict) Transaction(f func(ExtendedDictionary) bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	clone := &Dict{
		dict:  d.toMap(),
		def:   d.def,
		mutex: &sync.RWMutex{},
	}

	if ok := f(clone); ok {
		d.dict = clone.ToMap()
	}
}

func (d *Dict) IsMatch(f func(interface{}) bool) bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for _, v := range d.dict {
		if f(v) {
			return true
		}
	}

	return false
}

func (d *Dict) Match(f func(interface{}) bool) (interface{}, interface{}) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for k, v := range d.dict {
		if f(v) {
			return k, v
		}
	}

	return nil, nil
}

func (d *Dict) Foreach(f func(interface{}, interface{}) bool) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for k, v := range d.dict {
		if !f(k, v) {
			break
		}
	}
}

func (d *Dict) Iterator() (di Iterator) {
	di = &DictIterator{
		dict: d.dict,
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

func (di *DictIterator) init() {
	if !di.initd {
		di.initd = true
		di.kvchan = make(chan [2]interface{})

		go func() {
			defer func() {
				recover()
				close(di.kvchan)
			}()

			for k, v := range di.dict {
				if di.reset {
					break
				}
				var keyval = [2]interface{}{k, v}
				di.kvchan <- keyval
			}
		}()
	}
}

func (di *DictIterator) Next(val ...interface{}) bool {
	if !di.initd {
		di.init()
	}

	keyval, ok := <-di.kvchan

	if !ok {
		di.curkv = [2]interface{}{nil, nil}
		di.initd = false
		return false
	}

	di.curkv = keyval

	if lenval := len(val); lenval == 1 {
		util.MapValue(val[0], keyval[1])
	} else if lenval == 2 {
		util.MapValue(val[0], keyval[0])
		util.MapValue(val[1], keyval[1])
	}

	return true
}

func (di *DictIterator) Get() (interface{}, interface{}) {
	return di.curkv[0], di.curkv[1]
}

func (di *DictIterator) Reset() {
	di.reset = true
	_, _ = <-di.kvchan

	for true {
		_, ok := <-di.kvchan
		if !ok {
			break
		}
	}

	di.reset = false
}
