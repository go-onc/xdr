// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"reflect"
	"sync"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
	"go.e43.eu/xdr/internal/tags"
)

func newForT(t reflect.Type) func() interface{} {
	return func() interface{} {
		return reflect.New(t)
	}
}

type opaqueArrayCodec struct {
	bufs sync.Pool
	len  int
}

var _ xCodec = &opaqueArrayCodec{}

type arrayCodec struct {
	elem xCodec
	len  int
	size uintptr
}

func makeArrayCodec(cr *Coder, t reflect.Type, tag tags.XDRTag) xdrinterfaces.Codec {
	switch {
	case tag.Kind() != tags.Noop:
		return &errorCodec{errors.InvalidTagForTypeError{t, tag}}
	case tag.Next().Kind() == tags.Opaque:
		c := new(opaqueArrayCodec)
		c.bufs.New = newForT(t)
		c.len = t.Len()
		return c
	default:
		return &arrayCodec{
			elem: cr.getCodec(t.Elem(), tag.Next()),
			len:  t.Len(),
			size: t.Elem().Size(),
		}
	}
}

func (c *opaqueArrayCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	// If the user passed in an on-the-stack struct, e.g.
	// e.EncodeObject(struct{v byte[4] ``}), then v.CanAddr() may be false
	// which means we cannot slice it.
	//
	// In that scenario, we can either
	//   (1) Copy byte-by-byte, using v.Index(i) each time, or
	//   (2) Copy the data into a temporary buffer on the heap
	// We choose ot do (2):
	//   * In cases where the buffer is small, the memory overhead is
	//     likely to be low
	//   * In cases where the buffer is large, the overhead of going
	//     byte-by-byte through the value is likely to be considerable
	//
	// We amortise any allocation overhead if we hit this frequently by
	// storing these temporary buffers in a sync.Pool.
	//
	// We can't hit this case on decode because DecodeObject must always be
	// passed a pointer
	if !v.CanAddr() {
		p := c.bufs.Get().(reflect.Value)
		defer c.bufs.Put(p)

		e := p.Elem()
		e.Set(v)
		v = e
	}

	s := v.Slice(0, v.Len()).Bytes()
	return e.EncodeFixedOpaque(s)
}

func (c *opaqueArrayCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	l := v.Len()
	s := v.Slice(0, l).Bytes()
	return d.DecodeFixedOpaque(s)
}

func (c *arrayCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	for i, l := 0, v.Len(); i < l; i++ {
		if err := c.elem.Encode(e, v.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func (c *arrayCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	for i, l := 0, v.Len(); i < l; i++ {
		if err := c.elem.Decode(d, v.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

type opaqueSliceCodec struct {
	maxlen  int
	origMax uint32
}

var _ xCodec = &opaqueSliceCodec{}

type sliceCodec struct {
	elem    xCodec
	t       reflect.Type
	maxlen  int
	size    uintptr
	origMax uint32
}

func makeSliceCodec(cr *Coder, t reflect.Type, tag tags.XDRTag) xdrinterfaces.Codec {
	maxlen := ^uint32(0)

	switch tag.Kind() {
	case tags.MaxLen:
		maxlen = tag.OnlyValue()
	case tags.Noop:
		// Nothing
	default:
		return &errorCodec{errors.InvalidTagForTypeError{t, tag}}
	}

	// Cap lengths at maxInt
	origMax := maxlen
	if uint64(maxlen) > uint64(maxInt) {
		// Do two step assignment to prevent the compiler from being too smart
		// and complaining at us on builds where this code is unreachable
		i := maxInt
		maxlen = uint32(i)
	}

	switch {
	case tag.Next().Kind() == tags.Opaque:
		return &opaqueSliceCodec{int(maxlen), origMax}
	default:
		return &sliceCodec{
			elem:    cr.getCodec(t.Elem(), tag.Next()),
			t:       t,
			maxlen:  int(maxlen),
			size:    t.Elem().Size(),
			origMax: origMax,
		}
	}
}

func (c *opaqueSliceCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	s := v.Bytes()
	if len(s) > c.maxlen {
		return errors.LengthError{uint64(len(s)), uint64(c.origMax)}
	}

	return e.EncodeOpaque(s)
}

func (c *opaqueSliceCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	s, err := d.DecodeOpaque(c.maxlen)
	v.Set(reflect.ValueOf(s))
	return err
}

func (c *sliceCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	l := v.Len()
	if uint64(l) > uint64(c.maxlen) {
		return errors.LengthError{uint64(l), uint64(c.origMax)}
	}

	if err := e.EncodeUnsignedInt(uint32(l)); err != nil {
		return err
	}

	for i := 0; i < l; i++ {
		if err := c.elem.Encode(e, v.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func (c *sliceCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	l, err := d.DecodeUnsignedInt()
	switch {
	case err != nil:
		return err
	case l == 0:
		// Tiny opitmisation: Skip allocating zero-length slices
		v.Set(reflect.Zero(c.t))
		return nil
	case l > uint32(c.maxlen):
		return errors.LengthError{uint64(l), uint64(c.origMax)}
	}

	v.Set(reflect.MakeSlice(c.t, int(l), int(l)))

	for i := uint32(0); i < l; i++ {
		if err := c.elem.Decode(d, v.Index(int(i))); err != nil {
			return err
		}
	}
	return nil
}
