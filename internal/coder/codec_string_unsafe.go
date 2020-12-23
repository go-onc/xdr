// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// +build !nounsafe

package coder

import (
	"unsafe"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
)

func (c *fixedStringCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return c.encode(e, *(*string)(p))
}

func (c *fixedStringCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	s, err := d.DecodeFixedString(c.len)
	*(*string)(p) = s
	return err
}

func (c *varStringCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	return c.encode(e, *(*string)(p))
}

func (c *varStringCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	s, err := c.decode(d)
	*(*string)(p) = s
	return err
}
