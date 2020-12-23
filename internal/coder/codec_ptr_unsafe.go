// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// +build !nounsafe

package coder

import (
	"reflect"
	"unsafe"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
)

func (c *optCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	isNil := *(*uintptr)(p) == 0
	if err := e.EncodeBool(!isNil); err != nil {
		return err
	}

	if isNil {
		return nil
	}
	return c.elem.encodeUnsafe(e, p)
}

func (c *optCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	notNil, err := d.DecodeBool()
	if err != nil {
		return err
	} else if notNil {
		return c.elem.decodeUnsafe(d, p)
	}
	*(*uintptr)(p) = 0
	return nil
}

func (c *ptrCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	if v.IsNil() {
		return errors.ErrNilPointer
	}
	return c.elem.encodeUnsafe(e, unsafe.Pointer(v.Pointer()))
}

func (c *ptrCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	v.Set(reflect.New(c.elemt))
	return c.elem.decodeUnsafe(d, unsafe.Pointer(v.Pointer()))
}

func (c *ptrCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	if *(*uintptr)(p) == 0 {
		return errors.ErrNilPointer
	}
	return c.elem.encodeUnsafe(e, *(*unsafe.Pointer)(p))
}

func (c *ptrCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	v := unsafe.Pointer(reflect.New(c.elemt).Pointer())
	*(*unsafe.Pointer)(p) = v
	return c.elem.decodeUnsafe(d, v)
}
