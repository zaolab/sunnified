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

func (this *Stack) Len() int {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.len
}

func (this *Stack) Push(value interface{}) {
	s := &stackchain{
		data: value,
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.tail != nil {
		s.prev = this.tail
	}

	this.tail = s
	this.len++
}

func (this *Stack) Pop() (value interface{}) {
	value, _ = this.PopOk()
	return
}

func (this *Stack) PopOk() (value interface{}, ok bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.tail != nil {
		ok = true
		value = this.tail.data
		this.tail = this.tail.prev
		this.len--
	}

	return
}

func (this *Stack) Last() (value interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	if this.tail != nil {
		value = this.tail.data
	}
	return
}

func (this *Stack) Clear() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.tail = nil
	this.len = 0
}

func (this *Stack) ToSlice() (slice []interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	slice = make([]interface{}, this.len)
	curchain := this.tail

	for i := this.len - 1; i >= 0; i-- {
		slice[i] = curchain.data
		curchain = curchain.prev
	}

	return
}

func (this *Stack) String() string {
	return fmt.Sprintf("%v", this.ToSlice())
}

func (this *Stack) ToList() *List {
	return NewList(this.ToSlice()...)
}

func (this *Stack) Clone() (s *Stack) {
	s = NewStack()
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	curchain := this.tail
	s.tail = &stackchain{
		data: curchain.data,
	}
	newchain := s.tail

	for i := 0; i < this.len; i++ {
		curchain = curchain.prev
		schain := &stackchain{
			data: curchain.data,
		}
		newchain.prev = schain
		newchain = schain
	}

	return
}
