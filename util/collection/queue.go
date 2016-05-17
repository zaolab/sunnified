package collection

import (
	"fmt"
	"sync"

	"github.com/zaolab/sunnified/util"
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

func (q *Queue) Len() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return q.len
}

func (q *Queue) HasQueue() bool {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return q.len > 0
}

func (q *Queue) Pull() (val interface{}) {
	val, _ = q.PullOk()
	return
}

func (q *Queue) PullOk() (val interface{}, ok bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.head != nil {
		ok = true
		val = q.head.data
		q.head = q.head.next
		if q.head == nil {
			q.tail = nil
		}
		q.len--
	}

	return
}

func (q *Queue) PullDefault(def interface{}) interface{} {
	if val, ok := q.PullOk(); ok {
		return val
	}
	return def
}

func (q *Queue) Push(val ...interface{}) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	for _, v := range val {
		qc := &queuechain{
			data: v,
		}

		if q.head == nil {
			q.head = qc
		} else {
			q.tail.next = qc
		}

		q.tail = qc
		q.len++
	}
}

func (q *Queue) First() (val interface{}) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	if q.head != nil {
		val = q.head.data
	}

	return
}

func (q *Queue) Last() (val interface{}) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	if q.tail != nil {
		val = q.tail.data
	} else if q.head != nil {
		val = q.head.data
	}

	return
}

func (q *Queue) Clear() {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.head = nil
	q.tail = nil
	q.len = 0
}

func (q *Queue) ToSlice() []interface{} {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	slice := make([]interface{}, q.len)
	curqueue := q.head

	for i := 0; i < q.len; i++ {
		slice[i] = curqueue.data
		curqueue = curqueue.next
	}

	return slice
}

func (q *Queue) String() string {
	return fmt.Sprintf("%v", q.ToSlice())
}

func (q *Queue) ToList() *List {
	return NewList(q.ToSlice()...)
}

func (q *Queue) Clone() (qu *Queue) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	qu = NewQueue()
	curqueue := q.head

	for i := 0; i < q.len; i++ {
		qu.Push(curqueue.data)
	}

	return
}

func (q *Queue) Iterator() PopIterator {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return &QueueIterator{
		queue: q,
		seek:  q.head,
		mutex: q.mutex,
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

func (qi *QueueIterator) Next(val ...interface{}) (ok bool) {
	qi.mutex.RLock()
	defer qi.mutex.RUnlock()

	if qi.seek != nil {
		ok = true
		qi.curi++
		qi.curval = qi.seek.data
		qi.seek = qi.seek.next

		if lenval := len(val); lenval == 1 {
			util.MapValue(val[0], qi.curval)
		} else if lenval == 2 {
			util.MapValue(val[0], qi.curi)
			util.MapValue(val[1], qi.curval)
		}
	} else {
		qi.curi = -1
		qi.curval = nil
		qi.seek = qi.queue.head
	}

	return
}

func (qi *QueueIterator) PopNext(val ...interface{}) (ok bool) {
	if qi.curval, ok = qi.queue.PullOk(); ok {
		qi.curi++

		if lenval := len(val); lenval == 1 {
			util.MapValue(val[0], qi.curval)
		} else if lenval == 2 {
			util.MapValue(val[0], qi.curi)
			util.MapValue(val[1], qi.curval)
		}
	} else {
		qi.curi = -1
		qi.curval = nil
		qi.seek = nil
	}

	return
}

func (qi *QueueIterator) Get() (interface{}, interface{}) {
	return qi.curi, qi.curval
}

func (qi *QueueIterator) GetI() (int, interface{}) {
	return qi.curi, qi.curval
}

func (qi *QueueIterator) Reset() {
	qi.mutex.RLock()
	defer qi.mutex.RUnlock()
	qi.seek = qi.queue.head
}
