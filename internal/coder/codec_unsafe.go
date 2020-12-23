// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// +build !nounsafe

package coder

import (
	"reflect"
	"unsafe"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
)

// type codecUnsafe is like xdrinterfaces.Codec but with methods which take unsafe pointers
type codecUnsafe interface {
	xdrinterfaces.Codec

	// Encodes *p into the encoder e.
	encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error

	// Decodes *p from the decoder d.
	decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error
}

// type xCodec is the internal codec representation we use
// (in nounsafe builds it's aliased to xdrinterfaces.Codec)
type xCodec = codecUnsafe

// toXCodec makes a xdrinterfaces.Codec into an xCodec
// t should be the corresponding reflect.Type
func toXCodec(c xdrinterfaces.Codec, t reflect.Type) xCodec {
	switch c := c.(type) {
	case codecUnsafe:
		return c
	default:
		return &unsafeCodecWrapper{c, t}
	}
}

// toOriginalCodec makes an xCodec back into the original xdrinterfaces.Codec
func toOriginalCodec(x xCodec) xdrinterfaces.Codec {
	switch c := x.(type) {
	case *unsafeCodecWrapper:
		return c.Codec
	default:
		return c
	}
}

type unsafeCodecWrapper struct {
	xdrinterfaces.Codec
	t reflect.Type
}

func (w *unsafeCodecWrapper) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return w.Encode(e, reflect.NewAt(w.t, p).Elem())
}

func (w *unsafeCodecWrapper) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	return w.Decode(d, reflect.NewAt(w.t, p).Elem())
}
