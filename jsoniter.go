package main

import (
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/modern-go/reflect2"
)

var (
	jsonTwitter jsoniter.API
)

func init() {
	jsonTwitter = jsoniter.Config{
		EscapeHTML: false,
	}.Froze()

	jsonTwitter.RegisterExtension(new(jsoniterExtension))
}

type jsoniterExtension struct {
	jsoniter.DummyExtension
}

var (
	reflectTypeString    = reflect2.TypeOfPtr((*string)(nil)).Elem()
	reflectTypeTime      = reflect2.TypeOfPtr((*time.Time)(nil)).Elem()
	reflectTypeInterface = reflect2.TypeOfPtr((*interface{})(nil)).Elem()
)

func (ext jsoniterExtension) CreateEncoder(typ reflect2.Type) jsoniter.ValEncoder {
	switch typ {
	case reflectTypeString:
		return new(jsoniterStringEnc)
	case reflectTypeTime:
		return new(jsoniterTimeEncDec)
	}

	return nil
}
func (ext jsoniterExtension) CreateDecoder(typ reflect2.Type) jsoniter.ValDecoder {
	switch typ {
	case reflectTypeTime:
		return new(jsoniterTimeEncDec)

	case reflectTypeInterface:
		return new(jsoniterNumberDec)
	}

	return nil
}
