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

// Marshal marshals o into the returned buffer using DefaultCoder
func Marshal(o interface{}) ([]byte, error) {
	return DefaultCoder.Marshal(o)
}

// Unmarshal unmarshals buf into the object pointed to by op using DefaultCoder
func Unmarshal(buf []byte, op interface{}) error {
	return DefaultCoder.Unmarshal(buf, op)
}

// Write marshals o into the passed writer using DefaultCoder
func Write(w io.Writer, o interface{}) error {
	return DefaultCoder.Write(w, o)
}

// Read unmarshals *op out of the passed reader using DefaultCoder
func Read(r io.Reader, op interface{}) error {
	return DefaultCoder.Read(r, op)
}

// NewEncoder constructs a new encoder which writes to w using DefaultCoder
func NewEncoder(w io.Writer) Encoder {
	return DefaultCoder.NewEncoder(w)
}

// Constructs a new decoder which reads from r using DefaultCoder
func NewDecoder(r io.Reader) Decoder {
	return DefaultCoder.NewDecoder(r)
}

// NewCoder Construct a new Coder
func NewCoder() Coder {
	return coder.NewCoder()
}
