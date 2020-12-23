// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"bytes"
	"io"
	"math"
	"reflect"
	"sync"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
)

// 4 byte array which will always contain zeroes that we use whenever
// we need to emit padding
var pad [4]byte

var encoderPool = sync.Pool{
	New: func() interface{} {
		return &encoder{
			codecCacheSlot: 3,
		}
	},
}

type encoder struct {
	// Underlying writer
	w io.Writer
	// If the underlying writer is also an io.StringWriter, use that when writing
	// strings (to avoid allocs)
	ws io.StringWriter

	// Our coder
	cr *Coder

	// Small cache of most recently encoded types. Typically a small number of types
	// are repeatedly written to an encoder
	codecCache [4]struct {
		type_ reflect.Type
		codec xCodec
	}
	// Next slot for replacement
	codecCacheSlot int

	// Small scratch buffer (avoids needing to ever allocate when writing primitives)
	scratch [8]byte
}

var _ xdrinterfaces.Encoder = &encoder{}

func (e *encoder) reset(cr *Coder, w io.Writer) {
	e.w = w
	if ws, ok := w.(io.StringWriter); ok {
		e.ws = ws
	} else {
		e.ws = nil
	}

	if e.cr != cr {
		for i := range e.codecCache {
			e.codecCache[i].type_ = nil
			e.codecCache[i].codec = nil
		}
	}

	e.cr = cr
}

func (w *encoder) EncodeInt(i int32) error {
	w.scratch[0] = byte(i >> 24)
	w.scratch[1] = byte(i >> 16)
	w.scratch[2] = byte(i >> 8)
	w.scratch[3] = byte(i)
	_, err := w.w.Write(w.scratch[0:4])
	return err
}

func (w *encoder) EncodeUnsignedInt(i uint32) error {
	return w.EncodeInt(int32(i))
}

func (w *encoder) EncodeBool(b bool) error {
	i := 0
	if b {
		i = 1
	}
	return w.EncodeInt(int32(i))
}

func (w *encoder) EncodeHyper(i int64) error {
	w.scratch[0] = byte(i >> 56)
	w.scratch[1] = byte(i >> 48)
	w.scratch[2] = byte(i >> 40)
	w.scratch[3] = byte(i >> 32)
	w.scratch[4] = byte(i >> 24)
	w.scratch[5] = byte(i >> 16)
	w.scratch[6] = byte(i >> 8)
	w.scratch[7] = byte(i)
	_, err := w.w.Write(w.scratch[0:8])
	return err
}

func (w *encoder) EncodeUnsignedHyper(u uint64) error {
	return w.EncodeHyper(int64(u))
}

func (w *encoder) EncodeOpaque(buf []byte) error {
	if uint64(len(buf)) > uint64(math.MaxUint32) {
		return errors.LengthError{uint64(len(buf)), math.MaxUint32}
	}

	if err := w.EncodeUnsignedInt(uint32(len(buf))); err != nil {
		return err
	}
	return w.EncodeFixedOpaque(buf)
}

func (w *encoder) EncodeFixedOpaque(buf []byte) error {
	if _, err := w.w.Write(buf[:]); err != nil {
		return err
	}

	padding := (4 - (len(buf) & 3)) & 3
	_, err := w.w.Write(pad[0:padding])
	return err
}

func (w *encoder) EncodeString(s string) error {
	if uint64(len(s)) > uint64(math.MaxUint32) {
		return errors.LengthError{uint64(len(s)), math.MaxUint32}
	}

	if err := w.EncodeUnsignedInt(uint32(len(s))); err != nil {
		return err
	}
	return w.EncodeFixedString(s)
}

func (w *encoder) EncodeFixedString(s string) (err error) {
	if w.ws != nil {
		_, err = w.ws.WriteString(s)
	} else {
		_, err = w.w.Write([]byte(s))

	}
	if err != nil {
		return err
	}

	padding := (4 - (len(s) & 3)) & 3
	_, err = w.w.Write(pad[0:padding])
	return err
}

func (w *encoder) EncodeFloat(f float32) error {
	return w.EncodeUnsignedInt(math.Float32bits(f))
}

func (w *encoder) EncodeDouble(f float64) error {
	return w.EncodeUnsignedHyper(math.Float64bits(f))
}

func (w *encoder) Encode(o interface{}) error {
	return w.EncodeValue(reflect.ValueOf(o))
}

func (w *encoder) EncodeValue(v reflect.Value) error {
	t := v.Type()

	for _, e := range w.codecCache {
		if e.type_ == t {
			return e.codec.Encode(w, v)
		}
	}

	c := w.cr.getBaseCodec(t)
	w.codecCacheSlot = (w.codecCacheSlot + 1) & (len(w.codecCache) - 1)
	w.codecCache[w.codecCacheSlot].type_ = t
	w.codecCache[w.codecCacheSlot].codec = c

	return c.Encode(w, v)
}

func (w *encoder) release() {
	w.w = nil
	encoderPool.Put(w)
}

var marshalEncoderPool = sync.Pool{
	New: func() interface{} {
		me := &marshalEncoder{
			encoder: encoder{
				codecCacheSlot: 3,
			},
		}
		me.w = &me.b
		me.ws = &me.b
		return me
	},
}

type marshalEncoder struct {
	b bytes.Buffer
	encoder
}

func (e *marshalEncoder) reset(cr *Coder) {
	if e.cr != cr {
		for i := range e.codecCache {
			e.codecCache[i].type_ = nil
			e.codecCache[i].codec = nil
		}
	}

	e.cr = cr
}

func (e *marshalEncoder) release() {
	e.b.Reset()
	marshalEncoderPool.Put(e)
}
