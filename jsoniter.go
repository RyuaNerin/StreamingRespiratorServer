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

func init() {
	jsoniter.RegisterTypeDecoderFunc(
		"time.Time",
		func(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
			//(time.Time)(*ptr)
			str := iter.ReadString()
			t, err := time.Parse(RFC2822, str)
			if err == nil {
				*(*time.Time)(ptr) = t
			}
		},
	)
	jsoniter.RegisterTypeEncoderFunc(
		"time.Time",
		func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
			stream.WriteString((*(*time.Time)(ptr)).Format(RFC2822))
		},
		nil,
	)
}
