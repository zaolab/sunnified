package collection

import (
	"fmt"
	"github.com/zaolab/sunnified/util"
	"sync"
)

type queuechain struct {
	data interface{}
	next *queuechain
}

type Queue struct {
	head  *queuechain
	tail  *queuechain
	mutex *sync.RWMutex
	len   int
}

func NewQueue(data ...interface{}) (q *Queue) {
	q = &Queue{
		mutex: &sync.RWMutex{},
	}
	q.Push(data...)
	return
}

func (this *Queue) Len() int {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.len
}

func (this *Queue) HasQueue() bool {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.len > 0
}

func (this *Queue) Pull() (val interface{}) {
	val, _ = this.PullOk()
	return
}

func (this *Queue) PullOk() (val interface{}, ok bool) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.head != nil {
		ok = true
		val = this.head.data
		this.head = this.head.next
		if this.head == nil {
			this.tail = nil
		}
		this.len--
	}

	return
}

func (this *Queue) PullDefault(def interface{}) interface{} {
	if val, ok := this.PullOk(); ok {
		return val
	}
	return def
}

func (this *Queue) Push(val ...interface{}) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	for _, v := range val {
		q := &queuechain{
			data: v,
		}

		if this.head == nil {
			this.head = q
		} else {
			this.tail.next = q
		}

		this.tail = q
		this.len++
	}
}

func (this *Queue) First() (val interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	if this.head != nil {
		val = this.head.data
	}

	return
}

func (this *Queue) Last() (val interface{}) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	if this.tail != nil {
		val = this.tail.data
	} else if this.head != nil {
		val = this.head.data
	}

	return
}

func (this *Queue) Clear() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.head = nil
	this.tail = nil
	this.len = 0
}

func (this *Queue) ToSlice() []interface{} {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	slice := make([]interface{}, this.len)
	curqueue := this.head

	for i := 0; i < this.len; i++ {
		slice[i] = curqueue.data
		curqueue = curqueue.next
	}

	return slice
}

func (this *Queue) String() string {
	return fmt.Sprintf("%v", this.ToSlice())
}

func (this *Queue) ToList() *List {
	return NewList(this.ToSlice()...)
}

func (this *Queue) Clone() (q *Queue) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	q = NewQueue()
	curqueue := this.head

	for i := 0; i < this.len; i++ {
		q.Push(curqueue.data)
	}

	return
}

func (this *Queue) Iterator() PopIterator {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return &QueueIterator{
		queue: this,
		seek:  this.head,
		mutex: this.mutex,
		curi:  -1,
	}
}

type QueueIterator struct {
	queue  *Queue
	seek   *queuechain
	mutex  *sync.RWMutex
	curval interface{}
	curi   int
}

func (this *QueueIterator) Next(val ...interface{}) (ok bool) {
	this.mutex.RLock()
	defer this.mutex.RUnlock()

	if this.seek != nil {
		ok = true
		this.curi++
		this.curval = this.seek.data
		this.seek = this.seek.next

		if lenval := len(val); lenval == 1 {
			util.MapValue(val[0], this.curval)
		} else if lenval == 2 {
			util.MapValue(val[0], this.curi)
			util.MapValue(val[1], this.curval)
		}
	} else {
		this.curi = -1
		this.curval = nil
		this.seek = this.queue.head
	}

	return
}

func (this *QueueIterator) PopNext(val ...interface{}) (ok bool) {
	if this.curval, ok = this.queue.PullOk(); ok {
		this.curi++

		if lenval := len(val); lenval == 1 {
			util.MapValue(val[0], this.curval)
		} else if lenval == 2 {
			util.MapValue(val[0], this.curi)
			util.MapValue(val[1], this.curval)
		}
	} else {
		this.curi = -1
		this.curval = nil
		this.seek = nil
	}

	return
}

func (this *QueueIterator) Get() (interface{}, interface{}) {
	return this.curi, this.curval
}

func (this *QueueIterator) GetI() (int, interface{}) {
	return this.curi, this.curval
}

func (this *QueueIterator) Reset() {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	this.seek = this.queue.head
}
