// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"reflect"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
	"go.e43.eu/xdr/internal/tags"
)

type mapCodec struct {
	keyCodec   xCodec
	valueCodec xCodec
	t, kt, vt  reflect.Type
	maxlen     int
	origMax    uint32
}

func makeMapCodec(cr *Coder, t reflect.Type, tag tags.XDRTag) xdrinterfaces.Codec {
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

	return &mapCodec{
		keyCodec:   cr.getCodec(t.Key(), nil),
		valueCodec: cr.getCodec(t.Elem(), tag.Next()),
		t:          t,
		kt:         t.Key(),
		vt:         t.Elem(),
		maxlen:     int(maxlen),
		origMax:    origMax,
	}
}

func (c *mapCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	l := v.Len()
	if uint64(l) > uint64(c.maxlen) {
		return errors.LengthError{uint64(l), uint64(c.origMax)}
	}

	if err := e.EncodeUnsignedInt(uint32(l)); err != nil {
		return err
	}

	iter := v.MapRange()
	for iter.Next() {
		if err := c.keyCodec.Encode(e, iter.Key()); err != nil {
			return err
		}

		if err := c.valueCodec.Encode(e, iter.Value()); err != nil {
			return err
		}
	}
	return nil
}

func (c *mapCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	l, err := d.DecodeUnsignedInt()
	switch {
	case err != nil:
		return err
	case l > uint32(c.maxlen):
		return errors.LengthError{uint64(l), uint64(c.origMax)}
	}

	v.Set(reflect.MakeMapWithSize(c.t, int(l)))
	for i := uint32(0); i < l; i++ {
		kp := reflect.New(c.kt)
		vp := reflect.New(c.vt)

		k, vv := kp.Elem(), vp.Elem()

		if err := c.keyCodec.Decode(d, k); err != nil {
			return err
		}

		if err := c.valueCodec.Decode(d, vv); err != nil {
			return err
		}

		v.SetMapIndex(k, vv)
	}
	return nil
}
