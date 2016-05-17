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

func (s *Set) Get(index int) interface{} {
	return s.list.Get(index)
}

func (s *Set) MapValue(index int, value interface{}) (val interface{}) {
	return s.list.MapValue(index, value)
}

func (s *Set) First() interface{} {
	return s.list.First()
}

func (s *Set) Last() interface{} {
	return s.list.Last()
}

func (s *Set) Append(value interface{}) Array {
	s.list.mutex.Lock()
	defer s.list.mutex.Unlock()

	if _, exists := s.uniq[value]; !exists {
		s.list.append(value)
		s.uniq[value] = true
	}
	return s
}

func (s *Set) Len() int {
	return s.list.Len()
}

func (s *Set) Extend(values []interface{}) {
	s.list.mutex.Lock()
	defer s.list.mutex.Unlock()

	slice := make([]interface{}, 0, len(values))
	for _, val := range values {
		if _, exists := s.uniq[val]; !exists {
			slice = append(slice, val)
		}
	}

	s.list.extend(slice)
}

func (s *Set) ExtendArray(arr Array) {
	s.list.mutex.Lock()
	defer s.list.mutex.Unlock()

	slice := make([]interface{}, 0, arr.Len())
	var val interface{}
	for iter := arr.Iterator(); iter.Next(&val); {
		if _, exists := s.uniq[val]; !exists {
			slice = append(slice, val)
		}
	}

	s.list.extend(slice)
}

func (s *Set) ExtendList(list *List) {
	list.mutex.RLock()
	defer list.mutex.RUnlock()
	s.Extend(list.list[0:list.lenlist])
}

func (s *Set) ExtendSet(set *Set) {
	set.list.mutex.RLock()
	defer set.list.mutex.RUnlock()
	s.Extend(set.list.list[0:set.list.lenlist])
}

func (s *Set) Index(value interface{}) int {
	return s.list.Index(value)
}

func (s *Set) Indexes(value interface{}) []int {
	return s.list.Indexes(value)
}

func (s *Set) LastIndex(value interface{}) int {
	return s.list.LastIndex(value)
}

func (s *Set) Contains(value ...interface{}) bool {
	return s.list.Contains(value...)
}

func (s *Set) Set(index int, value interface{}) Array {
	s.list.mutex.Lock()
	defer s.list.mutex.Unlock()

	if _, exists := s.uniq[value]; !exists {
		s.list.set(index, value)
		s.uniq[value] = true
	}
	return s
}

func (s *Set) Insert(index int, value interface{}) Array {
	s.list.mutex.Lock()
	defer s.list.mutex.Unlock()

	if _, exists := s.uniq[value]; !exists {
		s.list.insert(index, value)
		s.uniq[value] = true
	}
	return s
}

func (s *Set) Pop() (val interface{}) {
	s.list.mutex.Lock()
	defer s.list.mutex.Unlock()
	val = s.list.pop()
	delete(s.uniq, val)
	return
}

func (s *Set) RemoveAt(index int) (val interface{}) {
	s.list.mutex.Lock()
	defer s.list.mutex.Unlock()

	val = s.list.removeat(index)
	delete(s.uniq, val)
	return
}

func (s *Set) Remove(value interface{}) {
	s.list.mutex.Lock()
	defer s.list.mutex.Unlock()

	s.list.removeat(s.list.index(value))
	delete(s.uniq, value)
}

func (s *Set) Clear() {
	s.list.mutex.Lock()
	defer s.list.mutex.Unlock()
	s.list.clear()
	s.uniq = make(map[interface{}]bool)
}

func (s *Set) Swap(x int, y int) {
	s.list.Swap(x, y)
}

func (s *Set) Reverse() {
	s.list.Reverse()
}

func (s *Set) Less(x int, y int) bool {
	return s.list.Less(x, y)
}

func (s *Set) Sort(f func(x interface{}, y interface{}) bool) {
	s.list.Sort(f)
}

func (s *Set) ToSlice() []interface{} {
	return s.list.ToSlice()
}

func (s *Set) ToList() *List {
	return s.list.Clone()
}

func (s *Set) String() string {
	return s.list.String()
}

func (s *Set) Clone() *Set {
	s.list.mutex.RLock()
	defer s.list.mutex.RUnlock()
	return s.clone()
}

func (s *Set) clone() (clone *Set) {
	clone = &Set{
		list: s.list.clone(),
		uniq: make(map[interface{}]bool),
	}

	for k, v := range s.uniq {
		clone.uniq[k] = v
	}

	return
}

func (s *Set) lock() {
	s.list.lock()
}

func (s *Set) unlock() {
	s.list.unlock()
}

func (s *Set) Transaction(f func(ExtendedArray) bool) {
	s.list.mutex.Lock()
	defer s.list.mutex.Unlock()

	clone := s.clone()

	if ok := f(clone); ok {
		clone.lock()
		defer clone.unlock()

		if clone.list.caplist != s.list.caplist {
			s.list.caplist = clone.list.caplist
			s.list.list = make([]interface{}, clone.list.caplist)
		}

		s.list.lenlist = clone.list.lenlist
		s.list.exlen = clone.list.exlen

		copy(s.list.list, clone.list.list[0:clone.list.lenlist])

		s.uniq = make(map[interface{}]bool)

		for k, v := range clone.uniq {
			s.uniq[k] = v
		}

	}
}

func (s *Set) Map(f func(interface{}) interface{}) ExtendedArray {
	return ExtendedArray(s.MapSet(f))
}

func (s *Set) MapSet(f func(interface{}) interface{}) (newset *Set) {
	s.list.mutex.RLock()
	defer s.list.mutex.RUnlock()

	newset = &Set{
		list: NewList(),
		uniq: make(map[interface{}]bool),
	}

	for i := 0; i < s.list.lenlist; i++ {
		newset.Insert(i, f(s.list.list[i]))
	}

	return
}

func (s *Set) Reduce(f func(interface{}, interface{}) interface{}, init interface{}) (value interface{}) {
	return s.list.Reduce(f, init)
}

func (s *Set) Foreach(f func(int, interface{}) bool) {
	s.list.Foreach(f)
}

func (s *Set) IsMatch(f func(interface{}) bool) bool {
	return s.list.IsMatch(f)
}

func (s *Set) Match(f func(interface{}) bool) (int, interface{}) {
	return s.list.Match(f)
}

func (s *Set) Filter(f func(interface{}) bool) ExtendedArray {
	return ExtendedArray(s.FilterSet(f))
}

func (s *Set) FilterSet(f func(interface{}) bool) (newset *Set) {
	newset = &Set{
		uniq: make(map[interface{}]bool),
	}

	newset.list = s.list.FilterList(func(val interface{}) bool {
		if f(val) {
			newset.uniq[val] = true
			return true
		}
		return false
	})

	return
}

func (s *Set) Iterator() NumIterator {
	return s.list.Iterator()
}
