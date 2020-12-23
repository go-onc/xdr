// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// +build nounsafe

package coder

import (
	"reflect"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
)

func (pc *ptrCodec) Encode(e xdrinterfaces.Encoder, v reflect.Value) error {
	if v.IsNil() {
		return errors.ErrNilPointer
	}
	return pc.elem.Encode(e, v.Elem())
}

func (pc *ptrCodec) Decode(d xdrinterfaces.Decoder, v reflect.Value) error {
	v.Set(reflect.New(pc.elemt))
	return pc.elem.Decode(d, v.Elem())
}
