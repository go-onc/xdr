// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// +build !nounsafe

package coder

import (
	"unsafe"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
)

func (dc *deferredCodec) encodeUnsafe(e xdrinterfaces.Encoder, p unsafe.Pointer) error {
	real := dc.real.Load()
	if real == nil {
		dc.wg.Wait()
		real = dc.real.Load()
	}
	return real.(xCodec).encodeUnsafe(e, p)
}

func (dc *deferredCodec) decodeUnsafe(d xdrinterfaces.Decoder, p unsafe.Pointer) error {
	real := dc.real.Load()
	if real == nil {
		dc.wg.Wait()
		real = dc.real.Load()
	}
	return real.(xCodec).decodeUnsafe(d, p)
}
