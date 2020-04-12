package main

import (
	"strconv"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

type jsoniterNumberDec struct{}

func (enc jsoniterNumberDec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.NumberValue:
		r := iter.ReadNumber()
		rs := r.String()

		if i64, err := strconv.ParseInt(rs, 10, 64); err == nil {
			*(*interface{})(ptr) = i64
			return
		}
		if ui64, err := strconv.ParseUint(rs, 10, 64); err == nil {
			*(*interface{})(ptr) = ui64
			return
		}
		if f64, err := strconv.ParseFloat(rs, 64); err == nil {
			*(*interface{})(ptr) = f64
			return
		}
		*(*interface{})(ptr) = r
	default:
		*(*interface{})(ptr) = iter.Read()
	}
}
