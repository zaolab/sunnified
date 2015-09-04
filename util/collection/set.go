package collection

import "github.com/zaolab/sunnified/util"

type Set struct {
	util.ValueGetter
	list *List
	uniq map[interface{}]bool
}

func NewSet(data ...interface{}) (li *Set) {
	li = &Set{
		list: NewList(),
		uniq: make(map[interface{}]bool),
	}

	li.ValueGetter = util.ValueGetter(li.list.getvalue)
	li.Extend(data)

	return
}

func (this *Set) Get(index int) interface{} {
	return this.list.Get(index)
}

func (this *Set) MapValue(index int, value interface{}) (val interface{}) {
	return this.list.MapValue(index, value)
}

func (this *Set) First() interface{} {
	return this.list.First()
}

func (this *Set) Last() interface{} {
	return this.list.Last()
}

func (this *Set) Append(value interface{}) Array {
	this.list.mutex.Lock()
	defer this.list.mutex.Unlock()

	if _, exists := this.uniq[value]; !exists {
		this.list.append(value)
		this.uniq[value] = true
	}
	return this
}

func (this *Set) Len() int {
	return this.list.Len()
}

func (this *Set) Extend(values []interface{}) {
	this.list.mutex.Lock()
	defer this.list.mutex.Unlock()

	slice := make([]interface{}, 0, len(values))
	for _, val := range values {
		if _, exists := this.uniq[val]; !exists {
			slice = append(slice, val)
		}
	}

	this.list.extend(slice)
}

func (this *Set) ExtendArray(arr Array) {
	this.list.mutex.Lock()
	defer this.list.mutex.Unlock()

	slice := make([]interface{}, 0, arr.Len())
	var val interface{}
	for iter := arr.Iterator(); iter.Next(&val); {
		if _, exists := this.uniq[val]; !exists {
			slice = append(slice, val)
		}
	}

	this.list.extend(slice)
}

func (this *Set) ExtendList(list *List) {
	list.mutex.RLock()
	defer list.mutex.RUnlock()
	this.Extend(list.list[0:list.lenlist])
}

func (this *Set) ExtendSet(set *Set) {
	set.list.mutex.RLock()
	defer set.list.mutex.RUnlock()
	this.Extend(set.list.list[0:set.list.lenlist])
}

func (this *Set) Index(value interface{}) int {
	return this.list.Index(value)
}

func (this *Set) Indexes(value interface{}) []int {
	return this.list.Indexes(value)
}

func (this *Set) LastIndex(value interface{}) int {
	return this.list.LastIndex(value)
}

func (this *Set) Contains(value ...interface{}) bool {
	return this.list.Contains(value...)
}

func (this *Set) Set(index int, value interface{}) Array {
	this.list.mutex.Lock()
	defer this.list.mutex.Unlock()

	if _, exists := this.uniq[value]; !exists {
		this.list.set(index, value)
		this.uniq[value] = true
	}
	return this
}

func (this *Set) Insert(index int, value interface{}) Array {
	this.list.mutex.Lock()
	defer this.list.mutex.Unlock()

	if _, exists := this.uniq[value]; !exists {
		this.list.insert(index, value)
		this.uniq[value] = true
	}
	return this
}

func (this *Set) Pop() (val interface{}) {
	this.list.mutex.Lock()
	defer this.list.mutex.Unlock()
	val = this.list.pop()
	delete(this.uniq, val)
	return
}

func (this *Set) RemoveAt(index int) (val interface{}) {
	this.list.mutex.Lock()
	defer this.list.mutex.Unlock()

	val = this.list.removeat(index)
	delete(this.uniq, val)
	return
}

func (this *Set) Remove(value interface{}) {
	this.list.mutex.Lock()
	defer this.list.mutex.Unlock()

	this.list.removeat(this.list.index(value))
	delete(this.uniq, value)
}

func (this *Set) Clear() {
	this.list.mutex.Lock()
	defer this.list.mutex.Unlock()
	this.list.clear()
	this.uniq = make(map[interface{}]bool)
}

func (this *Set) Swap(x int, y int) {
	this.list.Swap(x, y)
}

func (this *Set) Reverse() {
	this.list.Reverse()
}

func (this *Set) Less(x int, y int) bool {
	return this.list.Less(x, y)
}

func (this *Set) Sort(f func(x interface{}, y interface{}) bool) {
	this.list.Sort(f)
}

func (this *Set) ToSlice() []interface{} {
	return this.list.ToSlice()
}

func (this *Set) ToList() *List {
	return this.list.Clone()
}

func (this *Set) String() string {
	return this.list.String()
}

func (this *Set) Clone() *Set {
	this.list.mutex.RLock()
	defer this.list.mutex.RUnlock()
	return this.clone()
}

func (this *Set) clone() (clone *Set) {
	clone = &Set{
		list: this.list.clone(),
		uniq: make(map[interface{}]bool),
	}

	for k, v := range this.uniq {
		clone.uniq[k] = v
	}

	return
}

func (this *Set) lock() {
	this.list.lock()
}

func (this *Set) unlock() {
	this.list.unlock()
}

func (this *Set) Transaction(f func(ExtendedArray) bool) {
	this.list.mutex.Lock()
	defer this.list.mutex.Unlock()

	clone := this.clone()

	if ok := f(clone); ok {
		clone.lock()
		defer clone.unlock()

		if clone.list.caplist != this.list.caplist {
			this.list.caplist = clone.list.caplist
			this.list.list = make([]interface{}, clone.list.caplist)
		}

		this.list.lenlist = clone.list.lenlist
		this.list.exlen = clone.list.exlen

		copy(this.list.list, clone.list.list[0:clone.list.lenlist])

		this.uniq = make(map[interface{}]bool)

		for k, v := range clone.uniq {
			this.uniq[k] = v
		}

	}
}

func (this *Set) Map(f func(interface{}) interface{}) ExtendedArray {
	return ExtendedArray(this.MapSet(f))
}

func (this *Set) MapSet(f func(interface{}) interface{}) (newset *Set) {
	this.list.mutex.RLock()
	defer this.list.mutex.RUnlock()

	newset = &Set{
		list: NewList(),
		uniq: make(map[interface{}]bool),
	}

	for i := 0; i < this.list.lenlist; i++ {
		newset.Insert(i, f(this.list.list[i]))
	}

	return
}

func (this *Set) Reduce(f func(interface{}, interface{}) interface{}, init interface{}) (value interface{}) {
	return this.list.Reduce(f, init)
}

func (this *Set) Foreach(f func(int, interface{}) bool) {
	this.list.Foreach(f)
}

func (this *Set) IsMatch(f func(interface{}) bool) bool {
	return this.list.IsMatch(f)
}

func (this *Set) Match(f func(interface{}) bool) (int, interface{}) {
	return this.list.Match(f)
}

func (this *Set) Filter(f func(interface{}) bool) ExtendedArray {
	return ExtendedArray(this.FilterSet(f))
}

func (this *Set) FilterSet(f func(interface{}) bool) (newset *Set) {
	newset = &Set{
		uniq: make(map[interface{}]bool),
	}

	newset.list = this.list.FilterList(func(val interface{}) bool {
		if f(val) {
			newset.uniq[val] = true
			return true
		}
		return false
	})

	return
}

func (this *Set) Iterator() NumIterator {
	return this.list.Iterator()
}
