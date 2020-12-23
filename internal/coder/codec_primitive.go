// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"reflect"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
)

// boolCodec handles booleans
type boolCodec struct{}

var boolCodecI xCodec = boolCodec{}

func (_ boolCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return e.EncodeBool(v.Bool())
}

func (_ boolCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	b, e := d.DecodeBool()
	v.SetBool(b)
	return e
}

// [u]intCodec handle basic ([u]int8-[u]int32) integers
type intCodec struct{}
type uintCodec struct{}

func (_ intCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return e.EncodeInt(int32(v.Int()))
}

func (_ intCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	i, e := d.DecodeInt()
	v.SetInt(int64(i))
	return e
}

func (uc uintCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return e.EncodeUnsignedInt(uint32(v.Uint()))
}

func (uc uintCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	i, e := d.DecodeUnsignedInt()
	v.SetUint(uint64(i))
	return e
}

// [u]int[8/16/32]Codec specialise this codec to a specific size
type int8Codec struct{ intCodec }
type int16Codec struct{ intCodec }
type int32Codec struct{ intCodec }
type uint8Codec struct{ uintCodec }
type uint16Codec struct{ uintCodec }
type uint32Codec struct{ uintCodec }

var (
	int8CodecI   xCodec = int8Codec{}
	int16CodecI  xCodec = int16Codec{}
	int32CodecI  xCodec = int32Codec{}
	uint8CodecI  xCodec = uint8Codec{}
	uint16CodecI xCodec = uint16Codec{}
	uint32CodecI xCodec = uint32Codec{}
)

// [u]hyperCodec handles hyper ([u]int64) integers
type hyperCodec struct{}
type uhyperCodec struct{}

var (
	hyperCodecI  xCodec = hyperCodec{}
	uhyperCodecI xCodec = uhyperCodec{}
)

func (hc hyperCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return e.EncodeHyper(v.Int())
}

func (hc hyperCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	i, e := d.DecodeHyper()
	v.SetInt(i)
	return e
}

func (hc uhyperCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return e.EncodeUnsignedHyper(v.Uint())
}

func (hc uhyperCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	i, e := d.DecodeUnsignedHyper()
	v.SetUint(i)
	return e
}

// floatCodec handles floats
type floatCodec struct{}

var floatCodecI xCodec = floatCodec{}

func (_ floatCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return e.EncodeFloat(float32(v.Float()))
}

func (_ floatCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	f, e := d.DecodeFloat()
	v.SetFloat(float64(f))
	return e
}

// doubleCodec handles doubles
type doubleCodec struct{}

var doubleCodecI xCodec = doubleCodec{}

func (_ doubleCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return e.EncodeDouble(v.Float())
}

func (_ doubleCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	f, e := d.DecodeDouble()
	v.SetFloat(f)
	return e
}

// complex64Codec handles complex64s
type complex64Codec struct{}

var complex64CodecI xCodec = complex64Codec{}

func (_ complex64Codec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	c := complex64(v.Complex())
	if err := e.EncodeFloat(real(c)); err != nil {
		return err
	}
	return e.EncodeFloat(imag(c))
}

func (_ complex64Codec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	re, err := d.DecodeFloat()
	if err != nil {
		return err
	}
	im, err := d.DecodeFloat()
	if err != nil {
		return err
	}
	c := complex(re, im)
	v.SetComplex(complex128(c))
	return nil
}

// complex128Codec handles complex128s
type complex128Codec struct{}

var complex128CodecI xCodec = complex128Codec{}

func (_ complex128Codec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	c := v.Complex()
	if err := e.EncodeDouble(real(c)); err != nil {
		return err
	}
	return e.EncodeDouble(imag(c))
}

func (_ complex128Codec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	re, err := d.DecodeDouble()
	if err != nil {
		return err
	}
	im, err := d.DecodeDouble()
	if err != nil {
		return err
	}
	v.SetComplex(complex(re, im))
	return nil
}
