package view

import "sync/atomic"

const DefaultMinGZIPSize = 1000

var minGZIPSize = int32(DefaultMinGZIPSize)

func SetMinGZIPSize(i int) {
	atomic.StoreInt32(&minGZIPSize, int32(i))
}

func GetMinGZIPSize() int {
	i := atomic.LoadInt32(&minGZIPSize)
	return int(i)
}
