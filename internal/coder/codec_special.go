// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"reflect"
	"sync"
	"sync/atomic"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
)

// codec embedding a fixed, memoised error (generally
// indicating that a type can't be marshalled)
type errorCodec struct {
	err error
}

func (c *errorCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return c.err
}

func (c *errorCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	return c.err
}

// placeholder codec for types under construction, to handle cycles
type deferredCodec struct {
	real atomic.Value // xCodec
	wg   sync.WaitGroup
}

var _ xCodec = &deferredCodec{}

func newDeferredCodec() *deferredCodec {
	dc := new(deferredCodec)
	dc.wg.Add(1)
	return dc
}

func (dc *deferredCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	real := dc.real.Load()
	if real == nil {
		dc.wg.Wait()
		real = dc.real.Load()
	}
	return real.(xCodec).Encode(e, v)
}

func (dc *deferredCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	real := dc.real.Load()
	if real == nil {
		dc.wg.Wait()
		real = dc.real.Load()
	}
	return real.(xCodec).Decode(d, v)
}

func (dc *deferredCodec) resolve(real xCodec) {
	dc.real.Store(real)
	dc.wg.Done()
}

// marshalerCodec handles types which know how to self marshal
type marshalerCodec struct{}

var marshalerCodecI marshalerCodec

func (mc *marshalerCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	return v.Interface().(xdrinterfaces.Marshaler).MarshalXDR(e)
}

func (mc *marshalerCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	return v.Interface().(xdrinterfaces.Marshaler).UnmarshalXDR(d)
}
