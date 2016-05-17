package collection

import (
	"fmt"
	"sync"
)

type stackchain struct {
	data interface{}
	prev *stackchain
}

type Stack struct {
	tail  *stackchain
	mutex *sync.RWMutex
	len   int
}

func NewStack(data ...interface{}) (s *Stack) {
	s = &Stack{
		mutex: &sync.RWMutex{},
	}

	for _, val := range data {
		s.Push(val)
	}

	return
}

func (s *Stack) Len() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.len
}

func (s *Stack) Push(value interface{}) {
	sc := &stackchain{
		data: value,
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.tail != nil {
		sc.prev = s.tail
	}

	s.tail = sc
	s.len++
}

func (s *Stack) Pop() (value interface{}) {
	value, _ = s.PopOk()
	return
}

func (s *Stack) PopOk() (value interface{}, ok bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.tail != nil {
		ok = true
		value = s.tail.data
		s.tail = s.tail.prev
		s.len--
	}

	return
}

func (s *Stack) Last() (value interface{}) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if s.tail != nil {
		value = s.tail.data
	}
	return
}

func (s *Stack) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.tail = nil
	s.len = 0
}

func (s *Stack) ToSlice() (slice []interface{}) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	slice = make([]interface{}, s.len)
	curchain := s.tail

	for i := s.len - 1; i >= 0; i-- {
		slice[i] = curchain.data
		curchain = curchain.prev
	}

	return
}

func (s *Stack) String() string {
	return fmt.Sprintf("%v", s.ToSlice())
}

func (s *Stack) ToList() *List {
	return NewList(s.ToSlice()...)
}

func (s *Stack) Clone() (st *Stack) {
	st = NewStack()
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	curchain := s.tail
	st.tail = &stackchain{
		data: curchain.data,
	}
	newchain := st.tail

	for i := 0; i < s.len; i++ {
		curchain = curchain.prev
		schain := &stackchain{
			data: curchain.data,
		}
		newchain.prev = schain
		newchain = schain
	}

	return
}
