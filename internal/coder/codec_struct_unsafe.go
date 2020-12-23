// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// +build !nounsafe

package coder

import (
	"fmt"
	"reflect"
	"unsafe"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
	"go.e43.eu/xdr/internal/tags"
)

type field struct {
	index  int
	offset uintptr
	t      reflect.Type
	codec  xCodec
	name   string
}

func makeField(cr *Coder, f reflect.StructField, tag tags.XDRTag) field {
	if len(f.Index) != 1 {
		panic("Attempt to make field with index of depth >1")
	}

	return field{
		index:  f.Index[0],
		offset: f.Offset,
		t:      f.Type,
		codec:  cr.getCodec(f.Type, tag),
		name:   f.Name,
	}
}

func (f *field) encode(e xdrinterfaces.Encoder, p reflect.Value) (reflect.Value, error) {
	v := p.Field(f.index)
	err := f.codec.Encode(e, v)
	return v, err
}

func (f *field) encodeUnsafe(e xdrinterfaces.Encoder, pparent unsafe.Pointer) (unsafe.Pointer, error) {
	p := unsafe.Pointer(uintptr(pparent) + f.offset)
	err := f.codec.encodeUnsafe(e, p)
	return p, err
}

func (f *field) decode(d xdrinterfaces.Decoder, p reflect.Value) (reflect.Value, error) {
	v := p.Field(f.index)
	err := f.codec.Decode(d, v)
	return v, err
}

func (f *field) decodeUnsafe(d xdrinterfaces.Decoder, pparent unsafe.Pointer) (unsafe.Pointer, error) {
	p := unsafe.Pointer(uintptr(pparent) + f.offset)
	err := f.codec.decodeUnsafe(d, p)
	return p, err
}

func (c *structCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	if v.CanAddr() {
		return c.encodeUnsafe(e, unsafe.Pointer(v.UnsafeAddr()))
	} else {
		return c.encodeReflect(e, v)
	}
}

func (c *structCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	return c.decodeUnsafe(d, unsafe.Pointer(v.UnsafeAddr()))
}

func (c *structCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	for _, f := range c.fields {
		_, err := f.encodeUnsafe(e, p)
		if err != nil {
			return errors.WithFieldError(err, c.name, f.name)
		}
	}
	return nil
}

func (c *structCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	for _, f := range c.fields {
		_, err := f.decodeUnsafe(d, p)
		if err != nil {
			return errors.WithFieldError(err, c.name, f.name)
		}
	}
	return nil
}

func (c *unionCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	if v.CanAddr() {
		return c.encodeUnsafe(e, unsafe.Pointer(v.UnsafeAddr()))
	} else {
		return c.encodeReflect(e, v)
	}
}

func (c *unionCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	return c.decodeUnsafe(d, unsafe.Pointer(v.UnsafeAddr()))
}

func (c *unionCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	swp, err := c.switchField.encodeUnsafe(e, p)
	if err != nil {
		return errors.WithFieldError(err, c.name, c.switchField.name, "union:switch")
	}

	var swVal uint32
	switch c.switchKind {
	case switchKindBool:
		if *(*bool)(swp) {
			swVal = 1
		}
	default: // switchKindInt, switchKindUint
		// Go permits bitcasts by pointer aliasing
		swVal = *(*uint32)(swp)
	}

	caseField, exists := c.cases[swVal]
	if !exists {
		caseField = c.defaultCase
	}

	if caseField == -1 {
		err = errors.ErrUnionSwitchArmUndefined
		return errors.WithFieldError(err, c.name, "?", fmt.Sprintf("union:0x%x", caseField))
	}

	f := c.bodyFields[caseField]
	_, err = f.encodeUnsafe(e, p)
	if err != nil {
		return errors.WithFieldError(err, c.name, f.name, fmt.Sprintf("union:0x%x", swVal))
	}
	return nil
}

func (c *unionCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	swp, err := c.switchField.decodeUnsafe(d, p)
	if err != nil {
		return errors.WithFieldError(err, c.name, c.switchField.name, "union:switch")
	}

	var swVal uint32
	switch c.switchKind {
	case switchKindBool:
		if *(*bool)(swp) {
			swVal = 1
		}
	default: // switchKindInt, switchKindUint
		// Go permits bitcasts by pointer aliasing
		swVal = *(*uint32)(swp)
	}

	caseField, exists := c.cases[swVal]
	if !exists {
		caseField = c.defaultCase
	}

	if caseField == -1 {
		err = errors.ErrUnionSwitchArmUndefined
		return errors.WithFieldError(err, c.name, "?", fmt.Sprintf("union:0x%x", caseField))
	}

	f := c.bodyFields[caseField]
	_, err = f.decodeUnsafe(d, p)
	if err != nil {
		return errors.WithFieldError(err, c.name, f.name, fmt.Sprintf("union:0x%x", swVal))
	}
	return nil
}
