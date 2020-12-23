// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"fmt"
	"reflect"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
	"go.e43.eu/xdr/internal/tags"
)

// fixedStringCodec handles fixed strings,
// varStringCodec handles variable length strings
type fixedStringCodec struct {
	len int
}

type varStringCodec struct {
	maxlen  int
	origMax uint32
}

var _ xCodec = &fixedStringCodec{}
var _ xCodec = &varStringCodec{}

func makeStringCodec(t reflect.Type, tag tags.XDRTag) xdrinterfaces.Codec {
	if !tag.Next().Empty() {
		return &errorCodec{fmt.Errorf("string must not gave any following tags (%s)", tag)}
	}

	var (
		len   uint32
		fixed bool
	)

	switch tag.Kind() {
	case tags.Len:
		len = tag.OnlyValue()
		fixed = true

	case tags.MaxLen:
		len = tag.OnlyValue()

	case tags.Noop:
		len = ^uint32(0)

	default:
		return &errorCodec{errors.InvalidTagForTypeError{t, tag}}
	}

	origMax := len
	if uint64(len) > uint64(maxInt) {
		if fixed {
			// This can never work; reducing the maximum would be erroneous
			return &errorCodec{errors.LengthError{uint64(len), uint64(len)}}
		}

		// Do two step assignment to prevent the compiler from being too smart
		// and complaining at us on builds where this code is unreachable
		i := maxInt
		len = uint32(i)
	}

	if fixed {
		return &fixedStringCodec{int(len)}
	} else {
		return &varStringCodec{int(len), origMax}
	}
}

func (c *fixedStringCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return c.encode(e, v.String())
}

func (c *fixedStringCodec) encode(e xdrinterfaces.Encoder, s string) error {
	if uint64(len(s)) == uint64(c.len) {
		return e.EncodeFixedString(s)
	} else {
		return errors.ErrLengthIncorrect
	}
}

func (c *fixedStringCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	s, err := d.DecodeFixedString(c.len)
	v.SetString(s)
	return err
}

func (c *varStringCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return c.encode(e, v.String())
}

func (c *varStringCodec) encode(e xdrinterfaces.Encoder, s string) error {
	if uint64(len(s)) <= uint64(c.maxlen) {
		return e.EncodeString(s)
	} else {
		return errors.LengthError{uint64(len(s)), uint64(c.origMax)}
	}
}

func (c *varStringCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	s, err := c.decode(d)
	v.SetString(s)
	return err
}

func (c *varStringCodec) decode(d xdrinterfaces.Decoder) (string, error) {
	s, err := d.DecodeString(c.maxlen)
	if err != nil {
		if le, ok := err.(errors.LengthError); ok {
			le.Max = uint64(c.origMax)
			return s, le
		}
		return s, err
	}
	return s, nil
}
