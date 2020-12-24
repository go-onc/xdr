// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package xdr

import (
	"io"
	"reflect"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/coder"
)

type defaultCoder struct {
	coder.Coder
}

func (d *defaultCoder) RegisterCodec(template interface{}, c xdrinterfaces.Codec) {
	panic("Cannot register type on default codec")
}

func (d *defaultCoder) RegisterCodecReflect(type_ reflect.Type, c xdrinterfaces.Codec) {
	panic("Cannot register type on default codec")
}

// The default coder (used by the package global functions)
//
// This behaves identically to a coder created using NewCoder, except
// that it is not permitted to register any codecs upon it.
var DefaultCoder defaultCoder

// Marshals o into the returned buffer
func Marshal(o interface{}) ([]byte, error) {
	return DefaultCoder.Marshal(o)
}

// Unmarshals buf into the object pointed to by op
func Unmarshal(buf []byte, op interface{}) error {
	return DefaultCoder.Unmarshal(buf, op)
}

// Write marshals o into the passed writer
func Write(w io.Writer, o interface{}) error {
	return DefaultCoder.Write(w, o)
}

// Read unmarshals *op out of the passed reader
func Read(r io.Reader, op interface{}) error {
	return DefaultCoder.Read(r, op)
}

// Constructs a new encoder which writes to w
func NewEncoder(w io.Writer) Encoder {
	return DefaultCoder.NewEncoder(w)
}

// Constructs a new decoder which reads from r
func NewDecoder(r io.Reader) Decoder {
	return DefaultCoder.NewDecoder(r)
}

// Construct a new Coder
func NewCoder() Coder {
	return coder.NewCoder()
}
