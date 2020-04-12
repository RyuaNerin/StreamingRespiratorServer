package main

import (
	"time"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

const (
	// ddd MMM dd HH:mm:ss +ffff yyyy
	RFC2822 = "Mon Jan 02 15:04:05 -0700 2006"
)

type jsoniterTimeEncDec struct{}

func (enc jsoniterTimeEncDec) IsEmpty(ptr unsafe.Pointer) bool {
	return false
}
func (enc jsoniterTimeEncDec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	stream.WriteString((*(*time.Time)(ptr)).Format(RFC2822))
}
func (enc jsoniterTimeEncDec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	t, err := time.ParseInLocation(RFC2822, iter.ReadString(), time.UTC)
	if err != nil {
		iter.Error = err
		return
	}
	*((*time.Time)(ptr)) = t
}
