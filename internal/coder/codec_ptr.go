// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"reflect"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/tags"
)

// optCodec handles optional types (which must be pointerlike in Go)
type optCodec struct {
	elem xCodec
	nilp reflect.Value
}

func makeOptCodec(cr *Coder, t reflect.Type, tag tags.XDRTag) xCodec {
	// Strip the xt_opt and replace it with tag.Noop
	tag = tag.Next().Prepend(tags.Noop).Trimmed()

	return &optCodec{
		elem: cr.getCodec(t, tag),
		nilp: reflect.Zero(t),
	}
}

func (c *optCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	isNil := v.IsNil()
	err := e.EncodeBool(!isNil)
	if err != nil {
		return err
	}

	if isNil {
		return nil
	}

	return c.elem.Encode(e, v)
}

func (c *optCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	v.Set(c.nilp)

	isNonNil, err := d.DecodeBool()
	if err != nil {
		return err
	}

	if isNonNil {
		return c.elem.Decode(d, v)
	} else {
		v.Set(c.nilp)
		return nil
	}

}

// ptrCodec handles pointers
type ptrCodec struct {
	elem  xCodec
	elemt reflect.Type
}

func makePtrCodec(cr *Coder, t reflect.Type, tag tags.XDRTag) xCodec {
	elemt := t.Elem()
	c := cr.getCodec(elemt, tag.Next())
	return &ptrCodec{
		elem:  c,
		elemt: elemt,
	}
}
