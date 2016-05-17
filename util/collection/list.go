package collection

import (
	"fmt"
	"sort"
	"sync"

	"github.com/zaolab/sunnified/util"
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

func (l *List) getvalue(index interface{}) (interface{}, bool) {
	var i = index.(int)
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if l.lenlist > i {
		return l.list[i], true
	}
	return nil, false
}

func (l *List) MapValue(index int, value interface{}) (val interface{}) {
	if val = l.Get(index); val != nil && value != nil {
		util.MapValue(value, val)
	}
	return
}

func (l *List) First() interface{} {
	return l.Get(0)
}

func (l *List) Last() interface{} {
	return l.Get(l.lenlist)
}

func (l *List) Append(value interface{}) Array {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.append(value)
	return l
}

func (l *List) append(value interface{}) {
	if l.caplist == l.lenlist {
		l.expand(l.lenlist + 1)
	}

	l.list[l.lenlist] = value
	l.lenlist++
}

func (l *List) expand(length int) {
	// be sure calling method locks the list
	if l.lenlist < length {
		l.caplist = length + l.exlen
		if exlen := int(l.caplist / 10); exlen > l.exlen {
			l.exlen = exlen
		}
		tmplist := make([]interface{}, l.caplist)
		copy(tmplist, l.list[0:l.lenlist])
		l.list = tmplist
	}
}

func (l *List) Len() int {
	return l.lenlist
}

func (l *List) Extend(values []interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.extend(values)
}

func (l *List) extend(values []interface{}) {
	vallen := len(values)

	if vallen > 0 {
		finlen := l.prepareSize(vallen)

		copy(l.list[l.lenlist:finlen], values)
		l.lenlist = finlen
	}
}

func (l *List) prepareSize(size int) int {
	avalen := l.lenlist - l.caplist
	finlen := l.lenlist + size

	if avalen < size {
		l.expand(finlen)
	}

	return finlen
}

func (l *List) ExtendArray(arr Array) {
	if arrlen := arr.Len(); arrlen > 0 {
		l.mutex.Lock()
		defer l.mutex.Unlock()

		l.prepareSize(arrlen)

		var value interface{}
		for iter := arr.Iterator(); iter.Next(&value); {
			l.append(value)
		}
	}
}

func (l *List) ExtendList(list *List) {
	list.mutex.RLock()
	defer list.mutex.RUnlock()
	l.Extend(list.list[0:list.lenlist])
}

func (l *List) ExtendSet(set *Set) {
	set.list.mutex.RLock()
	defer set.list.mutex.RUnlock()
	l.Extend(set.list.list[0:set.list.lenlist])
}

func (l *List) Index(value interface{}) int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return l.index(value)
}

func (l *List) Indexes(value interface{}) (indexes []int) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	indexes = make([]int, 0, 2)

	for i, val := range l.list {
		if val == value {
			indexes = append(indexes, i)
		}
	}

	return
}

func (l *List) LastIndex(value interface{}) int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	for i := l.lenlist - 1; i >= 0; i-- {
		if l.list[i] == value {
			return i
		}
	}

	return -1
}

func (l *List) index(value interface{}) int {
	for i, val := range l.list {
		if val == value {
			return i
		}
	}

	return -1
}

func (l *List) Contains(value ...interface{}) bool {
	for _, val := range value {
		if l.Index(val) == -1 {
			return false
		}
	}
	return true
}

func (l *List) Set(index int, value interface{}) Array {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.set(index, value)
	return l
}

func (l *List) set(index int, value interface{}) {
	if index >= l.caplist {
		l.expand(index + 1)
		l.lenlist = index + 1
	} else if index >= l.lenlist {
		l.lenlist = index + 1
	}

	l.list[index] = value
}

func (l *List) Insert(index int, value interface{}) Array {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.insert(index, value)
	return l
}

func (l *List) insert(index int, value interface{}) {
	if index >= l.lenlist {
		l.set(index, value)
	} else {
		l.list = append(l.list[0:index], value, l.list[index:l.lenlist])
		l.lenlist++
	}
}

func (l *List) Pop() interface{} {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.pop()
}

func (l *List) pop() (val interface{}) {
	if l.lenlist > 0 {
		val = l.list[l.lenlist]
		l.list[l.lenlist] = nil
		l.lenlist--
	}

	return
}

func (l *List) RemoveAt(index int) interface{} {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.removeat(index)
}

func (l *List) removeat(index int) (val interface{}) {
	if index >= 0 && l.lenlist > index {
		val = l.list[index]
		newlist := make([]interface{}, l.caplist)
		copy(newlist, l.list[0:index])
		if l.lenlist > index+1 {
			copy(newlist[index:], l.list[index+1:l.lenlist])
		}
		l.list = newlist
		l.lenlist--
	}
	return
}

func (l *List) Remove(value interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.removeat(l.index(value))
}

func (l *List) Clear() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.clear()
}

func (l *List) clear() {
	l.lenlist = 0
	l.caplist = COMFORT_LEN
	l.exlen = COMFORT_LEN
	l.list = make([]interface{}, l.caplist)
}

func (l *List) Swap(x int, y int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.swap(x, y)
}

func (l *List) swap(x int, y int) {
	l.list[x], l.list[y] = l.list[y], l.list[x]
}

func (l *List) Reverse() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	backindex := l.lenlist - 1
	for i, rlen := 0, int(l.lenlist/2); i < rlen; i++ {
		l.swap(i, backindex)
		backindex--
	}
}

func (l *List) Less(x int, y int) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return Less(l.list[x], l.list[y])
}

func (l *List) Sort(less func(x interface{}, y interface{}) bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if less == nil {
		less = Less
	}

	sortlist := &SortList{
		list:    l.list,
		lenlist: l.lenlist,
		lessf:   less,
	}
	sort.Sort(sortlist)
}

func (l *List) ToSlice() []interface{} {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	tmplist := make([]interface{}, l.lenlist)
	copy(tmplist, l.list[0:l.lenlist])
	return tmplist
}

func (l *List) ToSet() (set *Set) {
	set = NewSet()
	set.ExtendList(l)
	return
}

func (l *List) String() string {
	return fmt.Sprintf("%v", l.list)
}

func (l *List) Clone() *List {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return l.clone()
}

func (l *List) clone() *List {
	clone := make([]interface{}, len(l.list))
	copy(clone, l.list[0:l.lenlist])

	return &List{
		list:    clone,
		mutex:   &sync.RWMutex{},
		lenlist: l.lenlist,
		caplist: l.caplist,
		exlen:   l.exlen,
	}
}

func (l *List) lock() {
	l.mutex.Lock()
}

func (l *List) unlock() {
	l.mutex.Unlock()
}

func (l *List) Transaction(f func(ExtendedArray) bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	clone := l.clone()

	if ok := f(clone); ok {
		clone.lock()
		defer clone.unlock()
		if clone.caplist != l.caplist {
			l.caplist = clone.caplist
			l.list = make([]interface{}, clone.caplist)
		}
		l.lenlist = clone.lenlist
		l.exlen = clone.exlen
		copy(l.list, clone.list[0:clone.lenlist])
	}
}

func (l *List) Foreach(f func(int, interface{}) bool) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	for i := 0; i < l.lenlist; i++ {
		if !f(i, l.list[i]) {
			break
		}
	}
}

func (l *List) IsMatch(f func(interface{}) bool) bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	for _, val := range l.list {
		if f(val) {
			return true
		}
	}

	return false
}

func (l *List) Match(f func(interface{}) bool) (int, interface{}) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	for i := 0; i < l.lenlist; i++ {
		if f(l.list[i]) {
			return i, l.list[i]
		}
	}

	return -1, nil
}

func (l *List) Map(f func(interface{}) interface{}) ExtendedArray {
	return ExtendedArray(l.MapList(f))
}

func (l *List) MapList(f func(interface{}) interface{}) (newlist *List) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	newlist = &List{
		list:    make([]interface{}, l.caplist),
		mutex:   &sync.RWMutex{},
		lenlist: l.lenlist,
		caplist: l.caplist,
		exlen:   l.exlen,
	}

	for i := 0; i < l.lenlist; i++ {
		newlist.list[i] = f(l.list[i])
	}

	return
}

func (l *List) Reduce(f func(interface{}, interface{}) interface{}, init interface{}) (value interface{}) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if l.lenlist > 0 {
		var i = 0

		if init != nil {
			value = init
		} else {
			value = l.list[0]
			i = 1
		}

		for ; i < l.lenlist; i++ {
			value = f(value, l.list[i])
		}
	} else if init != nil {
		value = init
	}

	return
}

func (l *List) Filter(f func(interface{}) bool) ExtendedArray {
	return ExtendedArray(l.FilterList(f))
}

func (l *List) FilterList(f func(interface{}) bool) (newlist *List) {
	l.mutex.RLock()
	var (
		exlen   = l.exlen
		caplist = l.caplist
		tmplist = make([]interface{}, 0, caplist)
	)
	l.mutex.RUnlock()

	l.Foreach(func(_ int, value interface{}) bool {
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

func (l *List) Iterator() NumIterator {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return &ListIterator{
		list:  l.list,
		len:   l.lenlist,
		mutex: l.mutex,
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

func (sl *SortList) Len() int {
	return sl.lenlist
}

func (sl *SortList) Less(x int, y int) bool {
	return sl.lessf(sl.list[x], sl.list[y])
}

func (sl *SortList) Swap(x int, y int) {
	sl.list[x], sl.list[y] = sl.list[y], sl.list[x]
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

func (li *ListIterator) Next(val ...interface{}) (ok bool) {
	defer func() {
		if err := recover(); err != nil {
			ok = false
			li.seek = 0
			li.mutex.RLock()
			li.len = len(li.list)
			li.mutex.RUnlock()
		}
	}()

	if li.seek < li.len {
		ok = true

		li.mutex.RLock()
		defer li.mutex.RUnlock()
		li.curi = li.seek
		li.curval = li.list[li.seek]

		if lenval := len(val); lenval == 1 {
			util.MapValue(val[0], li.curval)
		} else if lenval == 2 {
			util.MapValue(val[0], li.curi)
			util.MapValue(val[1], li.curval)
		}

		li.seek++
	} else {
		li.curi = -1
		li.curval = nil
		li.seek = 0
	}

	return
}

func (li *ListIterator) Get() (interface{}, interface{}) {
	return li.curi, li.curval
}

func (li *ListIterator) GetI() (int, interface{}) {
	return li.curi, li.curval
}

func (li *ListIterator) Reset() {
	li.seek = 0
}
