package main

import (
	"strconv"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
)

type JsoniterStringEscapeExtension struct {
	jsoniter.DummyExtension
}

var stringType = reflect2.TypeOfPtr((*string)(nil)).Elem()

func (ext *JsoniterStringEscapeExtension) CreateEncoder(typ reflect2.Type) jsoniter.ValEncoder {
	if typ == stringType {
		return new(JsoniterStringEscapeEncoder)
	}
	return nil
}

type JsoniterStringEscapeEncoder struct{}

func (enc JsoniterStringEscapeEncoder) IsEmpty(ptr unsafe.Pointer) bool {
	return len(*((*string)(ptr))) == 0
}
func (enc JsoniterStringEscapeEncoder) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	stream.SetBuffer(strconv.AppendQuoteToASCII(stream.Buffer(), *(*string)(ptr)))
}
