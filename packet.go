package main

import (
	"bytes"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

const (
	DefaultPakcetBufferSize = 16 * 1024 // 16 k
)

var (
	PacketPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, DefaultPakcetBufferSize))
		},
	}
)

type Packet struct {
	d []byte
	b *bytes.Buffer
}

func NewPacket(v interface{}) (p Packet, ok bool) {
	b := PacketPool.Get().(*bytes.Buffer)

	if err := jsoniter.NewEncoder(b).Encode(v); err != nil {
		return Packet{b.Bytes(), b}, true
	}

	PacketPool.Put(b)
	return p, false
}

func (p *Packet) Release() {
	PacketPool.Put(p.b)
}
