// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"io"
	"math"
	"reflect"
	"sync"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
)

var decoderPool = sync.Pool{
	New: func() interface{} {
		return new(decoder)
	},
}

type decoder struct {
	r  io.Reader
	cr *Coder
}

var _ xdrinterfaces.Decoder = &decoder{}

func (d *decoder) DecodeBool() (bool, error) {
	i, err := d.DecodeUnsignedInt()
	switch i {
	case 0:
		return false, err
	case 1:
		return true, err
	default:
		if err != nil {
			return false, err
		} else {
			return false, errors.ErrInvalidValue
		}
	}
}

func (d *decoder) DecodeInt() (int32, error) {
	u, err := d.DecodeUnsignedInt()
	return int32(u), err
}

func (d *decoder) DecodeUnsignedInt() (uint32, error) {
	var b [4]byte
	_, err := io.ReadFull(d.r, b[:])
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]), err
}

func (d *decoder) DecodeHyper() (int64, error) {
	u, err := d.DecodeUnsignedHyper()
	return int64(u), err
}

func (d *decoder) DecodeUnsignedHyper() (uint64, error) {
	var b [8]byte
	_, err := io.ReadFull(d.r, b[:])
	return (uint64(b[0])<<56 |
		uint64(b[1])<<48 |
		uint64(b[2])<<40 |
		uint64(b[3])<<32 |
		uint64(b[4])<<24 |
		uint64(b[5])<<16 |
		uint64(b[6])<<8 |
		uint64(b[7])), err
}

func (d *decoder) DecodeFloat() (float32, error) {
	i, err := d.DecodeUnsignedInt()
	return math.Float32frombits(i), err
}

func (d *decoder) DecodeDouble() (float64, error) {
	i, err := d.DecodeUnsignedHyper()
	return math.Float64frombits(i), err
}

func (d *decoder) OpaqueReader(maxLen uint32) (uint32, io.ReadCloser, error) {
	l, err := d.DecodeUnsignedInt()
	if err != nil {
		return 0, nil, err
	}

	if l > maxLen {
		return l, nil, errors.LengthError{uint64(l), uint64(maxLen)}
	}

	return l, d.FixedOpaqueReader(l), nil
}

func (d *decoder) FixedOpaqueReader(len uint32) io.ReadCloser {
	return newOpaqueReader(d.r, int64(len))
}

func (d *decoder) DecodeOpaque(maxLen int) ([]byte, error) {
	l, err := d.DecodeUnsignedInt()
	switch {
	case err != nil:
		return nil, err
	case l == 0:
		// Micro-optimisation: Just return buf when l==0, as there is nothing
		// for us to do.
		return nil, nil
	case uint64(l) > uint64(maxLen):
		return nil, errors.LengthError{uint64(l), uint64(maxLen)}
	}

	lPad := (int(l) + 3) & ^3
	buf := make([]byte, lPad)
	_, err = io.ReadFull(d.r, buf)
	return buf[0:int(l)], nil
}

func (d *decoder) DecodeFixedOpaque(buf []byte) error {
	var discard [4]byte

	n, err := io.ReadFull(d.r, buf)
	if err != nil {
		return err
	}

	// Discard any padding
	n = ((n + 3) & ^3) - n
	if n != 0 {
		n, err = io.ReadFull(d.r, discard[0:n])
	}
	return err
}

func (d *decoder) DecodeString(maxLen int) (string, error) {
	b, err := d.DecodeOpaque(maxLen)
	if err != nil {
		return "", err
	}
	return string(b), err
}

func (d *decoder) DecodeFixedString(len int) (string, error) {
	b := make([]byte, len)
	err := d.DecodeFixedOpaque(b)
	return string(b), err
}

func (d *decoder) Decode(op interface{}) (err error) {
	v := reflect.ValueOf(op)
	if v.Type().Kind() != reflect.Ptr {
		return errors.ErrNotPointer
	}

	return d.decodeValue(v.Elem())
}

func (d *decoder) DecodeValue(v reflect.Value) (err error) {
	if !v.CanSet() {
		return errors.ErrNotPointer
	}
	return d.decodeValue(v)
}

func (d *decoder) decodeValue(v reflect.Value) (err error) {
	return d.cr.getCodec(v.Type(), nil).Decode(d, v)
}

func (d *decoder) release() {
	d.r = nil
	d.cr = nil
	decoderPool.Put(d)
}
