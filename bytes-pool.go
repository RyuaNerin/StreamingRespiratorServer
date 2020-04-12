package main

import (
	"bytes"
	"sync"
)

const (
	DefaultBytesBufferSize = 16 * 1024 // 16 k
)

var (
	BytesPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, DefaultBytesBufferSize))
		},
	}
)
