package collection

import (
	"bytes"
	"fmt"
	"sort"
	"sync"

	"github.com/zaolab/sunnified/util"
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

func (od *OrderedDict) Len() int {
	od.mutex.RLock()
	defer od.mutex.RUnlock()
	return len(od.prevchain)
}

func (od *OrderedDict) getvalue(key interface{}) (interface{}, bool) {
	od.mutex.RLock()
	chain, ok := od.prevchain[key]
	od.mutex.RUnlock()

	if ok {
		return chain.next.value, true
	}

	return nil, false
}

func (od *OrderedDict) MapValue(key interface{}, value interface{}) (val interface{}) {
	if val = od.Get(key); val != nil && value != nil {
		util.MapValue(value, val)
	}
	return
}

func (od *OrderedDict) Set(key interface{}, value interface{}) Dictionary {
	od.mutex.Lock()
	defer od.mutex.Unlock()
	od.set(key, value)
	return od
}

func (od *OrderedDict) set(key interface{}, value interface{}) {
	if prevchain, exists := od.prevchain[key]; exists {
		prevchain.next.value = value
	} else {
		od.appendItem(key, value)
	}
}

func (od *OrderedDict) appendItem(key interface{}, value interface{}) {
	dchain := &datachain{
		key:   key,
		value: value,
	}
	var prevchain *datachain

	if od.tailchain == nil {
		od.tailchain = dchain
		od.headchain.next = dchain
		prevchain = od.headchain
	} else {
		od.tailchain.next = dchain
		prevchain = od.tailchain
		od.tailchain = dchain
	}

	od.prevchain[key] = prevchain
}

func (od *OrderedDict) SetDefault(key interface{}, value interface{}) Dictionary {
	od.mutex.Lock()
	defer od.mutex.Unlock()
	if _, exists := od.prevchain[key]; !exists {
		od.appendItem(key, value)
	}
	return od
}

func (od *OrderedDict) Keys() []interface{} {
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	return od.keys()
}

func (od *OrderedDict) keys() (keys []interface{}) {
	lendict := len(od.prevchain)
	keys = make([]interface{}, lendict)
	seekchain := od.headchain.next

	for i := 0; i < lendict; i++ {
		keys[i] = seekchain.key
		seekchain = seekchain.next
	}

	return
}

func (od *OrderedDict) Values() []interface{} {
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	return od.values()
}

func (od *OrderedDict) values() (values []interface{}) {
	lendict := len(od.prevchain)
	values = make([]interface{}, lendict)
	seekchain := od.headchain.next

	for i := 0; i < lendict; i++ {
		values[i] = seekchain.value
		seekchain = seekchain.next
	}

	return
}

func (od *OrderedDict) KeysValues() ([]interface{}, []interface{}) {
	od.mutex.RLock()
	defer od.mutex.RUnlock()
	return od.keysValues()
}

func (od *OrderedDict) keysValues() (keys []interface{}, values []interface{}) {
	lendict := len(od.prevchain)
	keys = make([]interface{}, lendict)
	values = make([]interface{}, lendict)
	seekchain := od.headchain.next

	for i := 0; i < lendict; i++ {
		keys[i] = seekchain.key
		values[i] = seekchain.value
		seekchain = seekchain.next
	}

	return
}

func (od *OrderedDict) Pairs() (pairs [][2]interface{}) {
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	lendict := len(od.prevchain)
	pairs = make([][2]interface{}, lendict)
	seekchain := od.headchain.next

	for i := 0; i < lendict; i++ {
		pairs[i] = [2]interface{}{seekchain.key, seekchain.value}
		seekchain = seekchain.next
	}

	return
}

func (od *OrderedDict) HasKey(key interface{}) (exists bool) {
	od.mutex.RLock()
	defer od.mutex.RUnlock()
	_, exists = od.prevchain[key]
	return
}

func (od *OrderedDict) Contains(values ...interface{}) bool {
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	for _, val := range values {
		if !od.hasValue(val) {
			return false
		}
	}

	return true
}

func (od *OrderedDict) hasValue(value interface{}) bool {
	/* TODO: perform benchmark
	however this is unordered
	for _, prevchain := range this.prevchain {
		if prevchain.next.value == value {
			return true
		}
	}
	*/
	for seek := od.headchain.next; seek != nil; seek = seek.next {
		if seek.value == value {
			return true
		}
	}
	return false
}

func (od *OrderedDict) KeyOf(value interface{}) interface{} {
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	for seek := od.headchain.next; seek != nil; seek = seek.next {
		if seek.value == value {
			return seek.key
		}
	}

	return nil
}

func (od *OrderedDict) KeysOf(value interface{}) (keys []interface{}) {
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	return od.keysOf(value)
}

func (od *OrderedDict) keysOf(value interface{}) (keys []interface{}) {
	keys = make([]interface{}, 0, 2)

	for seek := od.headchain.next; seek != nil; seek = seek.next {
		if seek.value == value {
			keys = append(keys, seek.key)
		}
	}

	return
}

func (od *OrderedDict) RemoveAt(key interface{}) interface{} {
	od.mutex.Lock()
	defer od.mutex.Unlock()
	return od.removeAt(key)
}

func (od *OrderedDict) removeAt(key interface{}) (value interface{}) {
	if prevchain, exists := od.prevchain[key]; exists {
		delete(od.prevchain, key)
		value = prevchain.next.value

		if prevchain.next.next == nil {
			od.tailchain = nil
		} else {
			od.prevchain[prevchain.next.next.key] = prevchain
			prevchain.next = prevchain.next.next
		}
	}
	return
}

func (od *OrderedDict) Remove(value interface{}) (keys []interface{}) {
	od.mutex.Lock()
	defer od.mutex.Unlock()
	keys = od.keysOf(value)
	for _, key := range keys {
		od.removeAt(key)
	}
	return
}

func (od *OrderedDict) Pop() (key interface{}, value interface{}) {
	od.mutex.Lock()
	defer od.mutex.Unlock()
	if od.tailchain != nil {
		key, value = od.tailchain.key, od.tailchain.value
		prevchain := od.prevchain[key]
		prevchain.next = nil

		delete(od.prevchain, key)

		if prevchain != od.headchain {
			od.tailchain = prevchain
		} else {
			od.tailchain = nil
		}
	}
	return
}

func (od *OrderedDict) Update(m map[interface{}]interface{}) {
	od.mutex.Lock()
	defer od.mutex.Unlock()
	for k, v := range m {
		od.set(k, v)
	}
}

func (od *OrderedDict) UpdateDictionary(d Dictionary) {
	od.mutex.Lock()
	defer od.mutex.Unlock()
	var key, value interface{}

	for iter := d.Iterator(); iter.Next(&key, &value); {
		od.set(key, value)
	}
}

func (od *OrderedDict) UpdateDict(d *Dict) {
	od.mutex.Lock()
	defer od.mutex.Unlock()
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for k, v := range d.dict {
		od.set(k, v)
	}
}

func (od *OrderedDict) UpdateOrderedDict(d *OrderedDict) {
	od.mutex.Lock()
	defer od.mutex.Unlock()
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for seek := d.headchain.next; seek != nil; seek = seek.next {
		od.set(seek.key, seek.value)
	}
}

func (od *OrderedDict) ToMap() map[interface{}]interface{} {
	od.mutex.RLock()
	defer od.mutex.RUnlock()
	return od.toMap()
}

func (od *OrderedDict) toMap() (m map[interface{}]interface{}) {
	m = make(map[interface{}]interface{})

	for seek := od.headchain.next; seek != nil; seek = seek.next {
		m[seek.key] = seek.value
	}

	return
}

func (od *OrderedDict) Clone() *OrderedDict {
	od.mutex.RLock()
	defer od.mutex.RUnlock()
	return od.clone()
}

func (od *OrderedDict) clone() *OrderedDict {
	var (
		m         = make(map[interface{}]*datachain)
		headchain = &datachain{}
		prevchain = headchain
	)

	for seek := od.headchain.next; seek != nil; seek = seek.next {
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
		def:       od.def,
		mutex:     &sync.RWMutex{},
	}
}

func (od *OrderedDict) String() string {
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	// 5 overhead bytes "map[]" + 2 inner bytes ": " + 2 min key+value
	b := make([]byte, 4, 5+(od.Len()*4))
	b[0] = 'm'
	b[1] = 'a'
	b[2] = 'p'
	b[3] = '['
	var buf = bytes.NewBuffer(b)

	for seek := od.headchain.next; seek != nil; seek = seek.next {
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

func (od *OrderedDict) lock() {
	od.mutex.Lock()
}

func (od *OrderedDict) unlock() {
	od.mutex.Unlock()
}

func (od *OrderedDict) Clear() {
	od.mutex.Lock()
	defer od.mutex.Unlock()
	od.prevchain = make(map[interface{}]*datachain)
	od.headchain.next = nil
	od.tailchain = nil
}

func (od *OrderedDict) Transaction(f func(ExtendedDictionary) bool) {
	od.mutex.Lock()
	defer od.mutex.Unlock()

	clone := od.clone()

	if ok := f(clone); ok {
		od.prevchain = clone.prevchain
		od.headchain = clone.headchain
		od.tailchain = clone.tailchain
	}
}

func (od *OrderedDict) Foreach(f func(interface{}, interface{}) bool) {
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	for seek := od.headchain.next; seek != nil; seek = seek.next {
		if !f(seek.key, seek.value) {
			break
		}
	}
}

func (od *OrderedDict) IsMatch(f func(interface{}) bool) bool {
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	for seek := od.headchain.next; seek != nil; seek = seek.next {
		if f(seek.value) {
			return true
		}
	}

	return false
}

func (od *OrderedDict) Match(f func(interface{}) bool) (interface{}, interface{}) {
	od.mutex.RLock()
	defer od.mutex.RUnlock()

	for seek := od.headchain.next; seek != nil; seek = seek.next {
		if f(seek.value) {
			return seek.key, seek.value
		}
	}
	return nil, nil
}

func (od *OrderedDict) Iterator() (di Iterator) {
	od.mutex.RLock()
	defer od.mutex.RUnlock()
	di = &OrderedDictIterator{
		headchain: od.headchain.next,
		mutex:     od.mutex,
	}
	return
}

func (od *OrderedDict) Reverse() {
	od.mutex.Lock()
	defer od.mutex.Unlock()

	lendict := len(od.prevchain)

	if lendict > 1 {
		curchain := od.tailchain
		prevchain := od.headchain

		for i := 0; i < lendict; i++ {
			thiskey := curchain.key
			prevchain.next = curchain
			od.prevchain[thiskey], curchain = prevchain, od.prevchain[thiskey]
			prevchain = prevchain.next
		}
	}
}

func (od *OrderedDict) Sort(less func(x interface{}, y interface{}) bool) {
	od.mutex.Lock()
	defer od.mutex.Unlock()

	if less == nil {
		less = Less
	}

	sortlist := &SortDictValue{
		lenlist: len(od.prevchain),
		lessf:   less,
	}
	sortlist.keys, sortlist.values = od.keysValues()

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

	od.prevchain = m
	od.headchain = headchain
	od.tailchain = prevchain
}

func (od *OrderedDict) SortByKey(less func(x interface{}, y interface{}) bool) {
	od.mutex.Lock()
	defer od.mutex.Unlock()

	if less == nil {
		less = Less
	}

	sortlist := &SortList{
		list:    od.keys(),
		lenlist: len(od.prevchain),
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
			value: od.prevchain[key].next.value,
		}
		m[key] = prevchain
		prevchain = prevchain.next
	}

	od.prevchain = m
	od.headchain = headchain
	od.tailchain = prevchain
}

type SortDictValue struct {
	values  []interface{}
	keys    []interface{}
	lenlist int
	lessf   func(x interface{}, y interface{}) bool
}

func (sd *SortDictValue) Len() int {
	return sd.lenlist
}

func (sd *SortDictValue) Less(x int, y int) bool {
	return sd.lessf(sd.values[x], sd.values[y])
}

func (sd *SortDictValue) Swap(x int, y int) {
	sd.values[x], sd.values[y] = sd.values[y], sd.values[x]
	sd.keys[x], sd.keys[y] = sd.keys[y], sd.keys[x]
}

type OrderedDictIterator struct {
	headchain *datachain
	mutex     *sync.RWMutex
	seek      *datachain
	key       interface{}
	value     interface{}
}

func (di *OrderedDictIterator) Next(val ...interface{}) (ok bool) {
	di.mutex.RLock()
	defer di.mutex.RUnlock()

	if di.seek == nil {
		di.seek = di.headchain
	} else {
		di.seek = di.seek.next
	}

	if ok = di.seek != nil; ok {
		di.key = di.seek.key
		di.value = di.seek.value

		if lenval := len(val); lenval == 1 {
			util.MapValue(val[0], di.value)
		} else if lenval == 2 {
			util.MapValue(val[0], di.key)
			util.MapValue(val[1], di.value)
		}
	} else {
		di.key = nil
		di.value = nil
	}

	return
}

func (di *OrderedDictIterator) Get() (interface{}, interface{}) {
	return di.key, di.value
}

func (di *OrderedDictIterator) Reset() {
	di.seek = nil
	di.key = nil
	di.value = nil
}
