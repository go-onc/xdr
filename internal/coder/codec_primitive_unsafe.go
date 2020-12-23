// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// +build !nounsafe

package coder

import (
	"unsafe"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
)

func (c boolCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeBool(*(*bool)(p))
}

func (c boolCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	b, err := d.DecodeBool()
	*(*bool)(p) = b
	return err
}

func (c int8Codec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeInt(int32(*(*int8)(p)))
}

func (c int8Codec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	i, err := d.DecodeInt()
	*(*int8)(p) = int8(i)
	return err
}

func (c int16Codec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeInt(int32(*(*int16)(p)))
}

func (c int16Codec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	i, err := d.DecodeInt()
	*(*int16)(p) = int16(i)
	return err
}

func (c int32Codec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeInt(*(*int32)(p))
}

func (c int32Codec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	i, err := d.DecodeInt()
	*(*int32)(p) = i
	return err
}

func (c uint8Codec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeUnsignedInt(uint32(*(*uint8)(p)))
}

func (c uint8Codec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	i, err := d.DecodeUnsignedInt()
	*(*uint8)(p) = uint8(i)
	return err
}

func (c uint16Codec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeUnsignedInt(uint32(*(*uint16)(p)))
}

func (c uint16Codec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	i, err := d.DecodeUnsignedInt()
	*(*uint16)(p) = uint16(i)
	return err
}

func (c uint32Codec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeUnsignedInt(*(*uint32)(p))
}

func (c uint32Codec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	i, err := d.DecodeUnsignedInt()
	*(*uint32)(p) = i
	return err
}

func (c hyperCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeHyper(*(*int64)(p))
}

func (c hyperCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	i, err := d.DecodeHyper()
	*(*int64)(p) = i
	return err
}

func (c uhyperCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeUnsignedHyper(*(*uint64)(p))
}

func (c uhyperCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	i, err := d.DecodeUnsignedHyper()
	*(*uint64)(p) = i
	return err
}

func (c floatCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeFloat(*(*float32)(p))
}

func (c floatCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	i, err := d.DecodeFloat()
	*(*float32)(p) = i
	return err
}

func (c doubleCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return e.EncodeDouble(*(*float64)(p))
}

func (c doubleCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	i, err := d.DecodeDouble()
	*(*float64)(p) = i
	return err
}

func (_ complex64Codec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	c := *(*complex64)(p)
	if err := e.EncodeFloat(real(c)); err != nil {
		return err
	}
	return e.EncodeFloat(imag(c))
}

func (_ complex64Codec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	re, err := d.DecodeFloat()
	if err != nil {
		return err
	}
	im, err := d.DecodeFloat()
	if err != nil {
		return err
	}
	*(*complex64)(p) = complex(re, im)
	return nil
}

func (_ complex128Codec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	c := *(*complex128)(p)
	if err := e.EncodeDouble(real(c)); err != nil {
		return err
	}
	return e.EncodeDouble(imag(c))
}

func (_ complex128Codec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	re, err := d.DecodeDouble()
	if err != nil {
		return err
	}
	im, err := d.DecodeDouble()
	if err != nil {
		return err
	}
	*(*complex128)(p) = complex(re, im)
	return nil
}
