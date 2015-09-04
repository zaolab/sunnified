package collection

import (
	"fmt"
	"github.com/zaolab/sunnified/util"
	"sort"
	"sync"
)

const COMFORT_LEN = 10

type List struct {
	util.ValueGetter
	list    []interface{}
	mutex   *sync.RWMutex
	lenlist int
	caplist int
	exlen   int
}

func NewList(data ...interface{}) (li *List) {
	li = &List{
		list:    make([]interface{}, COMFORT_LEN),
		mutex:   &sync.RWMutex{},
		caplist: COMFORT_LEN,
		exlen:   COMFORT_LEN,
	}

	li.ValueGetter = util.ValueGetter(li.getvalue)
	li.Extend(data)

	return
}

func (this *List) getvalue(index interface{}) (interface{}, bool) {
	var i = index.(int)
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	if this.lenlist > i {
		return this.list[i], true
	}
	return nil, false
}

func (this *List) MapValue(index int, value interface{}) (val interface{}) {
	if val = this.Get(index); val != nil && value != nil {
		util.MapValue(value, val)
	}
	return
}

func (this *List) First() interface{} {
	return this.Get(0)
}

func (this *List) Last() interface{} {
	return this.Get(this.lenlist)
}

func (this *List) Append(value interface{}) Array {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.append(value)
	return this
}

func (this *List) append(value interface{}) {
	if this.caplist == this.lenlist {
		this.expand(this.lenlist + 1)
	}

	this.list[this.lenlist] = value
	this.lenlist++
}

func (this *List) expand(length int) {
	// be sure calling method locks the list
	if this.lenlist < length {
		this.caplist = length + this.exlen
		if exlen := int(this.caplist / 10); exlen > this.exlen {
			this.exlen = exlen
		}
		tmplist := make([]interface{}, this.caplist)
		copy(tmplist, this.list[0:this.lenlist])
		this.list = tmplist
	}
}

func (this *List) Len() int {
	return this.lenlist
}

func (this *List) Extend(values []interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.extend(values)
}

func (this *List) extend(values []interface{}) {
	vallen := len(values)

	if vallen > 0 {
		finlen := this.prepareSize(vallen)

		copy(this.list[this.lenlist:finlen], values)
		this.lenlist = finlen
	}
}

func (this *List) prepareSize(size int) int {
	avalen := this.lenlist - this.caplist
	finlen := this.lenlist + size

	if avalen < size {
		this.expand(finlen)
	}

	return finlen
}

func (this *List) ExtendArray(arr Array) {
	if arrlen := arr.Len(); arrlen > 0 {
		this.mutex.Lock()
		defer this.mutex.Unlock()

		this.prepareSize(arrlen)

		var value interface{}
		for iter := arr.Iterator(); iter.Next(&value); {
			this.append(value)
		}
	}
}

func (this *List) ExtendList(list *List) {
	list.mutex.RLock()
	defer list.mutex.RUnlock()
	this.Extend(list.list[0:list.lenlist])
}

func (this *List) ExtendSet(set *Set) {
	set.list.mutex.RLock()
	defer set.list.mutex.RUnlock()
	this.Extend(set.list.list[0:set.list.lenlist])
}

func (this *List) Index(value interface{}) int {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	return this.index(value)
}

func (this *List) Indexes(value interface{}) (indexes []int) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	indexes = make([]int, 0, 2)

	for i, val := range this.list {
		if val == value {
			indexes = append(indexes, i)
		}
	}

	return
}

func (this *List) LastIndex(value interface{}) int {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for i := this.lenlist - 1; i >= 0; i-- {
		if this.list[i] == value {
			return i
		}
	}

	return -1
}

func (this *List) index(value interface{}) int {
	for i, val := range this.list {
		if val == value {
			return i
		}
	}

	return -1
}

func (this *List) Contains(value ...interface{}) bool {
	for _, val := range value {
		if this.Index(val) == -1 {
			return false
		}
	}
	return true
}

func (this *List) Set(index int, value interface{}) Array {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.set(index, value)
	return this
}

func (this *List) set(index int, value interface{}) {
	if index >= this.caplist {
		this.expand(index + 1)
		this.lenlist = index + 1
	} else if index >= this.lenlist {
		this.lenlist = index + 1
	}

	this.list[index] = value
}

func (this *List) Insert(index int, value interface{}) Array {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.insert(index, value)
	return this
}

func (this *List) insert(index int, value interface{}) {
	if index >= this.lenlist {
		this.set(index, value)
	} else {
		this.list = append(this.list[0:index], value, this.list[index:this.lenlist])
		this.lenlist++
	}
}

func (this *List) Pop() interface{} {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.pop()
}

func (this *List) pop() (val interface{}) {
	if this.lenlist > 0 {
		val = this.list[this.lenlist]
		this.list[this.lenlist] = nil
		this.lenlist--
	}

	return
}

func (this *List) RemoveAt(index int) interface{} {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.removeat(index)
}

func (this *List) removeat(index int) (val interface{}) {
	if index >= 0 && this.lenlist > index {
		val = this.list[index]
		newlist := make([]interface{}, this.caplist)
		copy(newlist, this.list[0:index])
		if this.lenlist > index+1 {
			copy(newlist[index:], this.list[index+1:this.lenlist])
		}
		this.list = newlist
		this.lenlist--
	}
	return
}

func (this *List) Remove(value interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	this.removeat(this.index(value))
}

func (this *List) Clear() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.clear()
}

func (this *List) clear() {
	this.lenlist = 0
	this.caplist = COMFORT_LEN
	this.exlen = COMFORT_LEN
	this.list = make([]interface{}, this.caplist)
}

func (this *List) Swap(x int, y int) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.swap(x, y)
}

func (this *List) swap(x int, y int) {
	this.list[x], this.list[y] = this.list[y], this.list[x]
}

func (this *List) Reverse() {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	backindex := this.lenlist - 1
	for i, rlen := 0, int(this.lenlist/2); i < rlen; i++ {
		this.swap(i, backindex)
		backindex--
	}
}

func (this *List) Less(x int, y int) bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return Less(this.list[x], this.list[y])
}

func (this *List) Sort(less func(x interface{}, y interface{}) bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if less == nil {
		less = Less
	}

	sortlist := &SortList{
		list:    this.list,
		lenlist: this.lenlist,
		lessf:   less,
	}
	sort.Sort(sortlist)
}

func (this *List) ToSlice() []interface{} {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	tmplist := make([]interface{}, this.lenlist)
	copy(tmplist, this.list[0:this.lenlist])
	return tmplist
}

func (this *List) ToSet() (set *Set) {
	set = NewSet()
	set.ExtendList(this)
	return
}

func (this *List) String() string {
	return fmt.Sprintf("%v", this.list)
}

func (this *List) Clone() *List {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.clone()
}

func (this *List) clone() *List {
	clone := make([]interface{}, len(this.list))
	copy(clone, this.list[0:this.lenlist])

	return &List{
		list:    clone,
		mutex:   &sync.RWMutex{},
		lenlist: this.lenlist,
		caplist: this.caplist,
		exlen:   this.exlen,
	}
}

func (this *List) lock() {
	this.mutex.Lock()
}

func (this *List) unlock() {
	this.mutex.Unlock()
}

func (this *List) Transaction(f func(ExtendedArray) bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	clone := this.clone()

	if ok := f(clone); ok {
		clone.lock()
		defer clone.unlock()
		if clone.caplist != this.caplist {
			this.caplist = clone.caplist
			this.list = make([]interface{}, clone.caplist)
		}
		this.lenlist = clone.lenlist
		this.exlen = clone.exlen
		copy(this.list, clone.list[0:clone.lenlist])
	}
}

func (this *List) Foreach(f func(int, interface{}) bool) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for i := 0; i < this.lenlist; i++ {
		if !f(i, this.list[i]) {
			break
		}
	}
}

func (this *List) IsMatch(f func(interface{}) bool) bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for _, val := range this.list {
		if f(val) {
			return true
		}
	}

	return false
}

func (this *List) Match(f func(interface{}) bool) (int, interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	for i := 0; i < this.lenlist; i++ {
		if f(this.list[i]) {
			return i, this.list[i]
		}
	}

	return -1, nil
}

func (this *List) Map(f func(interface{}) interface{}) ExtendedArray {
	return ExtendedArray(this.MapList(f))
}

func (this *List) MapList(f func(interface{}) interface{}) (newlist *List) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	newlist = &List{
		list:    make([]interface{}, this.caplist),
		mutex:   &sync.RWMutex{},
		lenlist: this.lenlist,
		caplist: this.caplist,
		exlen:   this.exlen,
	}

	for i := 0; i < this.lenlist; i++ {
		newlist.list[i] = f(this.list[i])
	}

	return
}

func (this *List) Reduce(f func(interface{}, interface{}) interface{}, init interface{}) (value interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	if this.lenlist > 0 {
		var i = 0

		if init != nil {
			value = init
		} else {
			value = this.list[0]
			i = 1
		}

		for ; i < this.lenlist; i++ {
			value = f(value, this.list[i])
		}
	} else if init != nil {
		value = init
	}

	return
}

func (this *List) Filter(f func(interface{}) bool) ExtendedArray {
	return ExtendedArray(this.FilterList(f))
}

func (this *List) FilterList(f func(interface{}) bool) (newlist *List) {
	this.mutex.RLock()
	var (
		exlen   = this.exlen
		caplist = this.caplist
		tmplist = make([]interface{}, 0, caplist)
	)
	this.mutex.RUnlock()

	this.Foreach(func(_ int, value interface{}) bool {
		if f(value) {
			tmplist = append(tmplist, value)
		}
		return true
	})

	lenlist := len(tmplist)
	return &List{
		list:    tmplist[0:caplist],
		lenlist: lenlist,
		caplist: caplist,
		exlen:   exlen,
		mutex:   &sync.RWMutex{},
	}
}

func (this *List) Iterator() NumIterator {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return &ListIterator{
		list:  this.list,
		len:   this.lenlist,
		mutex: this.mutex,
		curi:  -1,
	}
}

func Less(x interface{}, y interface{}) (lt bool) {
	switch valx := x.(type) {
	case int:
		lt = valx < y.(int)
	case int8:
		lt = valx < y.(int8)
	case int16:
		lt = valx < y.(int16)
	case int32:
		lt = valx < y.(int32)
	case int64:
		lt = valx < y.(int64)
	case uint:
		lt = valx < y.(uint)
	case uint8:
		lt = valx < y.(uint8)
	case uint16:
		lt = valx < y.(uint16)
	case uint32:
		lt = valx < y.(uint32)
	case uint64:
		lt = valx < y.(uint64)
	case float32:
		lt = valx < y.(float32)
	case float64:
		lt = valx < y.(float64)
	case string:
		lt = valx < y.(string)
	default:
		lt = fmt.Sprintf("%v", valx) < fmt.Sprintf("%v", y)
	}
	return
}

type SortList struct {
	list    []interface{}
	lenlist int
	lessf   func(x interface{}, y interface{}) bool
}

func (this *SortList) Len() int {
	return this.lenlist
}

func (this *SortList) Less(x int, y int) bool {
	return this.lessf(this.list[x], this.list[y])
}

func (this *SortList) Swap(x int, y int) {
	this.list[x], this.list[y] = this.list[y], this.list[x]
}

// usage
// for iter := li.Iterator(); iter.Next(&i, &value); {
//     ...
//     i, val := iter.GetI() // alternatively use this instead of .Next(&i, &value)
// }
type ListIterator struct {
	list   []interface{}
	len    int
	seek   int
	mutex  *sync.RWMutex
	curval interface{}
	curi   int
}

func (this *ListIterator) Next(val ...interface{}) (ok bool) {
	defer func() {
		if err := recover(); err != nil {
			ok = false
			this.seek = 0
			this.mutex.RLock()
			this.len = len(this.list)
			this.mutex.RUnlock()
		}
	}()

	if this.seek < this.len {
		ok = true

		this.mutex.RLock()
		defer this.mutex.RUnlock()
		this.curi = this.seek
		this.curval = this.list[this.seek]

		if lenval := len(val); lenval == 1 {
			util.MapValue(val[0], this.curval)
		} else if lenval == 2 {
			util.MapValue(val[0], this.curi)
			util.MapValue(val[1], this.curval)
		}

		this.seek++
	} else {
		this.curi = -1
		this.curval = nil
		this.seek = 0
	}

	return
}

func (this *ListIterator) Get() (interface{}, interface{}) {
	return this.curi, this.curval
}

func (this *ListIterator) GetI() (int, interface{}) {
	return this.curi, this.curval
}

func (this *ListIterator) Reset() {
	this.seek = 0
}
