// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// +build nounsafe

package coder

import (
	"reflect"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
)

// type xCodec is the internal codec representation we use
// (in !nounsafe builds it's different)
type xCodec = xdrinterfaces.Codec

func toXCodec(c xdrinterfaces.Codec, t reflect.Type) xCodec {
	return c
}

func toOriginalCodec(x xCodec) xdrinterfaces.Codec {
	return x
}
