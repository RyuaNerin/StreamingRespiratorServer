package main

import (
	"bytes"
	"io"
)

type Packet struct {
	d []byte
	b *bytes.Buffer
}

var (
	crLf = []byte("\r\n")
)

func newPacket(v interface{}) (p Packet, ok bool) {
	b := BytesPool.Get().(*bytes.Buffer)
	b.Reset()

	if err := jsonTwitter.NewEncoder(b).Encode(v); err != nil && err != io.EOF {
		BytesPool.Put(b)
		return p, false
	}
	b.Write(crLf)

	return Packet{b.Bytes(), b}, true
}
func newPacketFromReader(r io.Reader) (p Packet, ok bool) {
	b := BytesPool.Get().(*bytes.Buffer)
	b.Reset()

	if _, err := io.Copy(b, r); err == nil || err == io.EOF {
		BytesPool.Put(b)
		return p, false
	}
	b.Write(crLf)

	return Packet{b.Bytes(), b}, true
}

func (p *Packet) Release() {
	BytesPool.Put(p.b)
}
