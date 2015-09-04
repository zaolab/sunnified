package collection

import (
	"bytes"
	"fmt"
	"github.com/zaolab/sunnified/util"
	"sort"
	"sync"
)

type datachain struct {
	next  *datachain
	key   interface{}
	value interface{}
}

type OrderedDict struct {
	util.ValueGetter
	prevchain map[interface{}]*datachain
	def       interface{}
	mutex     *sync.RWMutex
	headchain *datachain
	tailchain *datachain
}

func NewOrderedDict(def interface{}, m ...map[interface{}]interface{}) (d *OrderedDict) {
	d = &OrderedDict{
		prevchain: make(map[interface{}]*datachain),
		def:       def,
		mutex:     &sync.RWMutex{},
		headchain: &datachain{},
	}
	d.ValueGetter = util.ValueGetter(d.getvalue)

	for _, mp := range m {
		d.Update(mp)
	}

	return
}

func (this *OrderedDict) Len() int {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return len(this.prevchain)
}

func (this *OrderedDict) getvalue(key interface{}) (interface{}, bool) {
	this.mutex.RLock()
	chain, ok := this.prevchain[key]
	this.mutex.RUnlock()

	if ok {
		return chain.next.value, true
	}

	return nil, false
}

func (this *OrderedDict) MapValue(key interface{}, value interface{}) (val interface{}) {
	if val = this.Get(key); val != nil && value != nil {
		util.MapValue(value, val)
	}
	return
}

func (this *OrderedDict) Set(key interface{}, value interface{}) Dictionary {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.set(key, value)
	return this
}

func (this *OrderedDict) set(key interface{}, value interface{}) {
	if prevchain, exists := this.prevchain[key]; exists {
		prevchain.next.value = value
	} else {
		this.appendItem(key, value)
	}
}

func (this *OrderedDict) appendItem(key interface{}, value interface{}) {
	dchain := &datachain{
		key:   key,
		value: value,
	}
	var prevchain *datachain

	if this.tailchain == nil {
		this.tailchain = dchain
		this.headchain.next = dchain
		prevchain = this.headchain
	} else {
		this.tailchain.next = dchain
		prevchain = this.tailchain
		this.tailchain = dchain
	}

	this.prevchain[key] = prevchain
}

func (this *OrderedDict) SetDefault(key interface{}, value interface{}) Dictionary {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if _, exists := this.prevchain[key]; !exists {
		this.appendItem(key, value)
	}
	return this
}

func (this *OrderedDict) Keys() []interface{} {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	return this.keys()
}

func (this *OrderedDict) keys() (keys []interface{}) {
	lendict := len(this.prevchain)
	keys = make([]interface{}, lendict)
	seekchain := this.headchain.next

	for i := 0; i < lendict; i++ {
		keys[i] = seekchain.key
		seekchain = seekchain.next
	}

	return
}

func (this *OrderedDict) Values() []interface{} {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	return this.values()
}

func (this *OrderedDict) values() (values []interface{}) {
	lendict := len(this.prevchain)
	values = make([]interface{}, lendict)
	seekchain := this.headchain.next

	for i := 0; i < lendict; i++ {
		values[i] = seekchain.value
		seekchain = seekchain.next
	}

	return
}

func (this *OrderedDict) KeysValues() ([]interface{}, []interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.keysValues()
}

func (this *OrderedDict) keysValues() (keys []interface{}, values []interface{}) {
	lendict := len(this.prevchain)
	keys = make([]interface{}, lendict)
	values = make([]interface{}, lendict)
	seekchain := this.headchain.next

	for i := 0; i < lendict; i++ {
		keys[i] = seekchain.key
		values[i] = seekchain.value
		seekchain = seekchain.next
	}

	return
}

func (this *OrderedDict) Pairs() (pairs [][2]interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	lendict := len(this.prevchain)
	pairs = make([][2]interface{}, lendict)
	seekchain := this.headchain.next

	for i := 0; i < lendict; i++ {
		pairs[i] = [2]interface{}{seekchain.key, seekchain.value}
		seekchain = seekchain.next
	}

	return
}

func (this *OrderedDict) HasKey(key interface{}) (exists bool) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	_, exists = this.prevchain[key]
	return
}

func (this *OrderedDict) Contains(values ...interface{}) bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for _, val := range values {
		if !this.hasValue(val) {
			return false
		}
	}

	return true
}

func (this *OrderedDict) hasValue(value interface{}) bool {
	/* TODO: perform benchmark
	however this is unordered
	for _, prevchain := range this.prevchain {
		if prevchain.next.value == value {
			return true
		}
	}
	*/
	for seek := this.headchain.next; seek != nil; seek = seek.next {
		if seek.value == value {
			return true
		}
	}
	return false
}

func (this *OrderedDict) KeyOf(value interface{}) interface{} {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for seek := this.headchain.next; seek != nil; seek = seek.next {
		if seek.value == value {
			return seek.key
		}
	}

	return nil
}

func (this *OrderedDict) KeysOf(value interface{}) (keys []interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	return this.keysOf(value)
}

func (this *OrderedDict) keysOf(value interface{}) (keys []interface{}) {
	keys = make([]interface{}, 0, 2)

	for seek := this.headchain.next; seek != nil; seek = seek.next {
		if seek.value == value {
			keys = append(keys, seek.key)
		}
	}

	return
}

func (this *OrderedDict) RemoveAt(key interface{}) interface{} {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.removeAt(key)
}

func (this *OrderedDict) removeAt(key interface{}) (value interface{}) {
	if prevchain, exists := this.prevchain[key]; exists {
		delete(this.prevchain, key)
		value = prevchain.next.value

		if prevchain.next.next == nil {
			this.tailchain = nil
		} else {
			this.prevchain[prevchain.next.next.key] = prevchain
			prevchain.next = prevchain.next.next
		}
	}
	return
}

func (this *OrderedDict) Remove(value interface{}) (keys []interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	keys = this.keysOf(value)
	for _, key := range keys {
		this.removeAt(key)
	}
	return
}

func (this *OrderedDict) Pop() (key interface{}, value interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	if this.tailchain != nil {
		key, value = this.tailchain.key, this.tailchain.value
		prevchain := this.prevchain[key]
		prevchain.next = nil

		delete(this.prevchain, key)

		if prevchain != this.headchain {
			this.tailchain = prevchain
		} else {
			this.tailchain = nil
		}
	}
	return
}

func (this *OrderedDict) Update(m map[interface{}]interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	for k, v := range m {
		this.set(k, v)
	}
}

func (this *OrderedDict) UpdateDictionary(d Dictionary) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	var key, value interface{}

	for iter := d.Iterator(); iter.Next(&key, &value); {
		this.set(key, value)
	}
}

func (this *OrderedDict) UpdateDict(d *Dict) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for k, v := range d.dict {
		this.set(k, v)
	}
}

func (this *OrderedDict) UpdateOrderedDict(d *OrderedDict) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for seek := d.headchain.next; seek != nil; seek = seek.next {
		this.set(seek.key, seek.value)
	}
}

func (this *OrderedDict) ToMap() map[interface{}]interface{} {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.toMap()
}

func (this *OrderedDict) toMap() (m map[interface{}]interface{}) {
	m = make(map[interface{}]interface{})

	for seek := this.headchain.next; seek != nil; seek = seek.next {
		m[seek.key] = seek.value
	}

	return
}

func (this *OrderedDict) Clone() *OrderedDict {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.clone()
}

func (this *OrderedDict) clone() *OrderedDict {
	var (
		m         = make(map[interface{}]*datachain)
		headchain = &datachain{}
		prevchain = headchain
	)

	for seek := this.headchain.next; seek != nil; seek = seek.next {
		prevchain.next = &datachain{
			key:   seek.key,
			value: seek.value,
		}
		m[seek.key] = prevchain
		prevchain = prevchain.next
	}

	return &OrderedDict{
		prevchain: m,
		headchain: headchain,
		tailchain: prevchain,
		def:       this.def,
		mutex:     &sync.RWMutex{},
	}
}

func (this *OrderedDict) String() string {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	// 5 overhead bytes "map[]" + 2 inner bytes ": " + 2 min key+value
	b := make([]byte, 4, 5+(this.Len()*4))
	b[0] = 'm'
	b[1] = 'a'
	b[2] = 'p'
	b[3] = '['
	var buf = bytes.NewBuffer(b)

	for seek := this.headchain.next; seek != nil; seek = seek.next {
		fmt.Fprintf(buf, "%v", seek.key)
		buf.WriteByte(':')
		fmt.Fprintf(buf, "%v", seek.value)
		if seek.next != nil {
			buf.WriteByte(' ')
		}
	}

	buf.WriteByte(']')
	return buf.String()
}

func (this *OrderedDict) lock() {
	this.mutex.Lock()
}

func (this *OrderedDict) unlock() {
	this.mutex.Unlock()
}

func (this *OrderedDict) Clear() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.prevchain = make(map[interface{}]*datachain)
	this.headchain.next = nil
	this.tailchain = nil
}

func (this *OrderedDict) Transaction(f func(ExtendedDictionary) bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	clone := this.clone()

	if ok := f(clone); ok {
		this.prevchain = clone.prevchain
		this.headchain = clone.headchain
		this.tailchain = clone.tailchain
	}
}

func (this *OrderedDict) Foreach(f func(interface{}, interface{}) bool) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for seek := this.headchain.next; seek != nil; seek = seek.next {
		if !f(seek.key, seek.value) {
			break
		}
	}
}

func (this *OrderedDict) IsMatch(f func(interface{}) bool) bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for seek := this.headchain.next; seek != nil; seek = seek.next {
		if f(seek.value) {
			return true
		}
	}

	return false
}

func (this *OrderedDict) Match(f func(interface{}) bool) (interface{}, interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for seek := this.headchain.next; seek != nil; seek = seek.next {
		if f(seek.value) {
			return seek.key, seek.value
		}
	}
	return nil, nil
}

func (this *OrderedDict) Iterator() (di Iterator) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	di = &OrderedDictIterator{
		headchain: this.headchain.next,
		mutex:     this.mutex,
	}
	return
}

func (this *OrderedDict) Reverse() {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	lendict := len(this.prevchain)

	if lendict > 1 {
		curchain := this.tailchain
		prevchain := this.headchain

		for i := 0; i < lendict; i++ {
			thiskey := curchain.key
			prevchain.next = curchain
			this.prevchain[thiskey], curchain = prevchain, this.prevchain[thiskey]
			prevchain = prevchain.next
		}
	}
}

func (this *OrderedDict) Sort(less func(x interface{}, y interface{}) bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if less == nil {
		less = Less
	}

	sortlist := &SortDictValue{
		lenlist: len(this.prevchain),
		lessf:   less,
	}
	sortlist.keys, sortlist.values = this.keysValues()

	sort.Sort(sortlist)

	var (
		m         = make(map[interface{}]*datachain)
		headchain = &datachain{}
		prevchain = headchain
	)

	for i, key := range sortlist.keys {
		prevchain.next = &datachain{
			key:   key,
			value: sortlist.values[i],
		}
		m[key] = prevchain
		prevchain = prevchain.next
	}

	this.prevchain = m
	this.headchain = headchain
	this.tailchain = prevchain
}

func (this *OrderedDict) SortByKey(less func(x interface{}, y interface{}) bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if less == nil {
		less = Less
	}

	sortlist := &SortList{
		list:    this.keys(),
		lenlist: len(this.prevchain),
		lessf:   less,
	}

	sort.Sort(sortlist)

	var (
		m         = make(map[interface{}]*datachain)
		headchain = &datachain{}
		prevchain = headchain
	)

	for _, key := range sortlist.list {
		prevchain.next = &datachain{
			key:   key,
			value: this.prevchain[key].next.value,
		}
		m[key] = prevchain
		prevchain = prevchain.next
	}

	this.prevchain = m
	this.headchain = headchain
	this.tailchain = prevchain
}

type SortDictValue struct {
	values  []interface{}
	keys    []interface{}
	lenlist int
	lessf   func(x interface{}, y interface{}) bool
}

func (this *SortDictValue) Len() int {
	return this.lenlist
}

func (this *SortDictValue) Less(x int, y int) bool {
	return this.lessf(this.values[x], this.values[y])
}

func (this *SortDictValue) Swap(x int, y int) {
	this.values[x], this.values[y] = this.values[y], this.values[x]
	this.keys[x], this.keys[y] = this.keys[y], this.keys[x]
}

type OrderedDictIterator struct {
	headchain *datachain
	mutex     *sync.RWMutex
	seek      *datachain
	key       interface{}
	value     interface{}
}

func (this *OrderedDictIterator) Next(val ...interface{}) (ok bool) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	if this.seek == nil {
		this.seek = this.headchain
	} else {
		this.seek = this.seek.next
	}

	if ok = this.seek != nil; ok {
		this.key = this.seek.key
		this.value = this.seek.value

		if lenval := len(val); lenval == 1 {
			util.MapValue(val[0], this.value)
		} else if lenval == 2 {
			util.MapValue(val[0], this.key)
			util.MapValue(val[1], this.value)
		}
	} else {
		this.key = nil
		this.value = nil
	}

	return
}

func (this *OrderedDictIterator) Get() (interface{}, interface{}) {
	return this.key, this.value
}

func (this *OrderedDictIterator) Reset() {
	this.seek = nil
	this.key = nil
	this.value = nil
}
