// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// +build nounsafe

package coder

import (
	"fmt"
	"reflect"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
	"go.e43.eu/xdr/internal/tags"
)

type field struct {
	index int
	codec xCodec
	name  string
}

func makeField(cr *Coder, f reflect.StructField, tag tags.XDRTag) field {
	if len(f.Index) != 1 {
		panic("Attempt to make field with index of depth >1")
	}

	return field{
		index: f.Index[0],
		codec: cr.getCodec(f.Type, tag),
		name:  f.Name,
	}
}

func (f *field) encode(e xdrinterfaces.Encoder, p reflect.Value) (reflect.Value, error) {
	v := p.Field(f.index)
	err := f.codec.Encode(e, v)
	return v, err
}

func (f *field) decode(d xdrinterfaces.Decoder, p reflect.Value) (reflect.Value, error) {
	v := p.Field(f.index)
	err := f.codec.Decode(d, v)
	return v, err
}

func (c *structCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return c.encodeReflect(e, v)
}

func (c *structCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	for _, f := range c.fields {
		_, err := f.decode(d, v)
		if err != nil {
			return errors.WithFieldError(err, c.name, f.name)
		}
	}
	return nil
}

func (c *unionCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return c.encodeReflect(e, v)
}

func (c *unionCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) (err error) {
	swv, err := c.switchField.decode(d, v)
	if err != nil {
		err = errors.WithFieldError(err, c.name, c.switchField.name, "union:switch")
		return
	}

	var swVal uint32
	switch c.switchKind {
	case switchKindBool:
		if swv.Bool() {
			swVal = 1
		}
	case switchKindUint:
		swVal = uint32(swv.Uint())
	default: //switchKindInt
		swVal = uint32(swv.Int())
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
	_, err = f.decode(d, v)
	if err != nil {
		err = errors.WithFieldError(err, c.name, f.name, fmt.Sprintf("union:0x%x", swVal))
	}
	return
}
