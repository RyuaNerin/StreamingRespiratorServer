package main

import (
	"bytes"
	"sync"

	jsoniter "github.com/json-iterator/go"
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

func Serialize(v interface{}) (data []byte, buff *bytes.Buffer) {
	buff = PoolBytesBuffer.Get().(*bytes.Buffer)

	if err := jsoniter.NewEncoder(buff).Encode(v); err != nil {
		return buff.Bytes(), buff
	}

	PoolBytesBuffer.Put(buff)
	return nil, nil
}
