package main

import (
	"strconv"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

type jsoniterStringEnc struct{}

func (enc jsoniterStringEnc) IsEmpty(ptr unsafe.Pointer) bool {
	return len(*((*string)(ptr))) == 0
}
func (enc jsoniterStringEnc) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	stream.SetBuffer(strconv.AppendQuoteToASCII(stream.Buffer(), *(*string)(ptr)))
}
