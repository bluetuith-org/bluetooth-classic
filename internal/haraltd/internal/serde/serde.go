//go:build !linux && haraltd

package serde

import (
	"sync"

	"github.com/ugorji/go/codec"
)

// resolver holds an encoder and decoder.
type resolver struct {
	check bool

	jsonEncoder *codec.Encoder
	jsonDecoder *codec.Decoder
	jsonHandle  codec.JsonHandle

	jsonData []byte

	jsonMu sync.Mutex
}

var gendecoder resolver

func init() {
	if !gendecoder.check {
		gendecoder.jsonHandle = codec.JsonHandle{}
		gendecoder.jsonHandle.TypeInfos = codec.NewTypeInfos([]string{"json"})
		gendecoder.jsonEncoder = codec.NewEncoderBytes(&gendecoder.jsonData, &gendecoder.jsonHandle)
		gendecoder.jsonDecoder = codec.NewDecoderBytes(gendecoder.jsonData, &gendecoder.jsonHandle)

		gendecoder.jsonData = make([]byte, 0, 8192)
		gendecoder.check = true
	}
}

// MarshalJSON marshals a value of a specific type to UTF-8 bytes.
func MarshalJSON[T any](v T) ([]byte, error) {
	gendecoder.jsonMu.Lock()
	defer gendecoder.jsonMu.Unlock()

	gendecoder.jsonEncoder.ResetBytes(&gendecoder.jsonData)

	// copy the slice
	return gendecoder.jsonData, gendecoder.jsonEncoder.Encode(v)
}

// UnmarshalJSON unmarshals the provided JSON as bytes to the value of a specific type.
func UnmarshalJSON[T any](data []byte, marshalTo T) error {
	gendecoder.jsonMu.Lock()
	defer gendecoder.jsonMu.Unlock()

	gendecoder.jsonDecoder.ResetBytes(data)

	return gendecoder.jsonDecoder.Decode(marshalTo)
}
