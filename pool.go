package main

import (
	"bytes"
	"sync"
)

const (
	DefaultBytesBufferPoolSize = 16 * 1024 // 16 k
)

var (
	PoolBytesBuffer = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, DefaultBytesBufferPoolSize))
		},
	}
)
