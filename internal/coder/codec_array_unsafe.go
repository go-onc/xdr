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

func (c *opaqueArrayCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	// Construct a []byte of length c.lem pointing at *p
	var slice []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	hdr.Data = uintptr(p)
	hdr.Cap = c.len
	hdr.Len = c.len
	return e.EncodeFixedOpaque(slice)
}

func (c *opaqueArrayCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	// Construct a []byte of length c.lem pointing at *p
	var slice []byte
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
	hdr.Data = uintptr(p)
	hdr.Cap = c.len
	hdr.Len = c.len
	return d.DecodeFixedOpaque(slice)
}

func (c *arrayCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	for i := 0; i < c.len; i++ {
		if err := c.elem.encodeUnsafe(e, unsafe.Pointer(uintptr(p)+uintptr(i)*c.size)); err != nil {
			return err
		}
	}
	return nil
}

func (c *arrayCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	for i := 0; i < c.len; i++ {
		if err := c.elem.decodeUnsafe(d, unsafe.Pointer(uintptr(p)+uintptr(i)*c.size)); err != nil {
			return err
		}
	}
	return nil
}

func (c *opaqueSliceCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	s := *(*[]byte)(p)
	if len(s) > c.maxlen {
		return errors.LengthError{uint64(len(s)), uint64(c.origMax)}
	}

	return e.EncodeOpaque(s)
}

func (c *opaqueSliceCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	s, err := d.DecodeOpaque(c.maxlen)
	*(*[]byte)(p) = s
	return err
}

func (c *sliceCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	sh := ((*reflect.SliceHeader)(p))
	if uint64(sh.Len) > uint64(c.maxlen) {
		return errors.LengthError{uint64(sh.Len), uint64(c.origMax)}
	}

	if err := e.EncodeUnsignedInt(uint32(sh.Len)); err != nil {
		return err
	}

	pd := unsafe.Pointer(sh.Data)
	for i := 0; i < sh.Len; i++ {
		if err := c.elem.encodeUnsafe(e, unsafe.Pointer(uintptr(pd)+uintptr(i)*c.size)); err != nil {
			return err
		}
	}
	return nil
}

func (c *sliceCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	l, err := d.DecodeUnsignedInt()
	switch {
	case err != nil:
		return err
	case l == 0:
		// Tiny opitmisation: Skip allocating zero-length slices
		sh := ((*reflect.SliceHeader)(p))
		sh.Len = 0
		sh.Cap = 0
		sh.Data = 0
		return nil
	case l > uint32(c.maxlen):
		return errors.LengthError{uint64(l), uint64(c.origMax)}
	}

	rp := reflect.NewAt(c.t, p)
	rp.Elem().Set(reflect.MakeSlice(c.t, int(l), int(l)))

	sh := ((*reflect.SliceHeader)(p))
	pd := unsafe.Pointer(sh.Data)
	for i := 0; i < sh.Len; i++ {
		if err := c.elem.decodeUnsafe(d, unsafe.Pointer(uintptr(pd)+uintptr(i)*c.size)); err != nil {
			return err
		}
	}
	return nil
}
