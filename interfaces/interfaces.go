// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// Package xdrinterfaces defines the primary interfaces of the XDR encoder
//
// (This package is primarily separated out in order to permit the implementation to
// be broken down into multiple packages)
package xdrinterfaces

import (
	"io"
	"reflect"
)

// interface Marshaler is the interface implemented by a type which knows how to encode
// and decode itself to/froms XDR
type Marshaler interface {
	MarshalXDR(e Encoder) error
	UnmarshalXDR(d Decoder) error
}

// interface Codec is the interface by which the marshalling of types which are
// not natively supported may be defined.
//
// Codecs may be registered with a Coder in order to specify how to handle a
// specific type.
//
// It is recommended to use a custom Marshaler (or `xdr` struct tags) implementation
// when defining your own types instead of defining a Codec. However, this may be useful
// when dealing with third party types.
type Codec interface {
	// Encodes v into the encoder e.
	Encode(e Encoder, v reflect.Value) error

	// Decodes v from the decoder d.
	Decode(d Decoder, v reflect.Value) error
}

// interface Coder is the top-level interface to the XDR library
//
// A coder (which may be safely used from multiple threads) provides the ability
// to marshal objects to and from XDR. It also contains a repository of Codecs
// which know how to marshal various types
type Coder interface {
	// Marshals o into the returned buffer
	Marshal(o interface{}) ([]byte, error)

	// Unmarshals buf into the object pointed to by op
	Unmarshal(buf []byte, op interface{}) error

	// Write marshals o into the passed writer
	Write(w io.Writer, o interface{}) error

	// Read unmarshals *op out of the passed reader
	Read(r io.Reader, op interface{}) error

	// Constructs a new encoder which writes to w
	NewEncoder(w io.Writer) Encoder

	// Constructs a new decoder which reads from r
	NewDecoder(r io.Reader) Decoder

	// Registers the codec. Panics if a codec is already registered for
	// the type, or an attempt is made to register a codec for a type
	// for which it is not permitted to register codecs.
	RegisterCodec(template interface{}, c Codec)
	RegisterCodecReflect(type_ reflect.Type, c Codec)
}

// interface Encoder is the interface to the XDR encoder
type Encoder interface {
	// EncodeBool writes a bool to the XDR encoder
	EncodeBool(b bool) error

	// EncodeInt writes an int to the XDR encoder
	EncodeInt(i int32) error

	// EncodeUnsignedInt writes an unsigned int to the XDR encoder
	EncodeUnsignedInt(i uint32) error

	// EncodeHyper writes a hyper (int64) to the XDR encoder
	EncodeHyper(h int64) error

	// EncodeUnsignedHyper writes an unsigned hyper (uint64) to the XDR encoder
	EncodeUnsignedHyper(h uint64) error

	// EncodeFloat writes a single precision floating point number to the XDR encoder
	EncodeFloat(f float32) error

	// EncodeDouble writes a double precision floating point number to the XDR encoder
	EncodeDouble(d float64) error

	// EncodeOpaque writes an `opaque` (dense byte slice) to the XDR encoder
	EncodeOpaque(b []byte) error

	// EncodeFixedOpaque writes a fixed length opaque (dense byte slice) to the XDR encoder
	// This is for fixed length fields; no length prefix will be written
	EncodeFixedOpaque(b []byte) error

	// EncodeString writes a string to the XDR encoder
	EncodeString(s string) error

	// EncodeFixedString writes a fixed length string to the XDR encoder
	EncodeFixedString(s string) error

	// Encode writes an object to the XDR encoder
	Encode(o interface{}) error

	// EncodeValue encodes an object to the XDR encoder (via reflection)
	EncodeValue(v reflect.Value) error
}

// interface Decoder is the interface to the XDR decoder
type Decoder interface {
	DecodeBool() (bool, error)
	DecodeInt() (int32, error)
	DecodeUnsignedInt() (uint32, error)
	DecodeHyper() (int64, error)
	DecodeUnsignedHyper() (uint64, error)

	// DecodeFloat reads a single precision floating point number from the XDR decoder
	DecodeFloat() (float32, error)

	// DecodeDouble reads a double precision floating point number from the XDR decoder
	DecodeDouble() (float64, error)

	// DecodeOpaque reads an opaque of maximum length maxLen from the XDR decoder
	// A newly allocated buffer is returned.
	DecodeOpaque(maxLen int) ([]byte, error)

	// OpaqueReader returns an io.Reader which reads the body of the opaque from the
	// XDR decoder.
	//
	// The stream *must* be closed before reading further. It is not sufficient
	// to just exhaust the stream; Close() must also be called to consume padding.
	OpaqueReader(maxLen uint32) (uint32, io.ReadCloser, error)

	// DecodeFixedOpaque reads a fixed-size opaque into the passed buffer
	DecodeFixedOpaque(buf []byte) error

	// FixedOpaqueReader returns an io.Reader which reads the body of the fixed-length
	// opaque of length len in the same fasihion as OpaqueReader()
	FixedOpaqueReader(len uint32) io.ReadCloser

	// ReadString reads a string (with maximum length maxLen) from the decoder
	DecodeString(maxLen int) (string, error)

	// DecodeFixedString reads a fixed length string (of length len) from the decoder
	DecodeFixedString(len int) (string, error)

	// Decode reads an object from the stream into *op.
	Decode(op interface{}) error

	// DecodeValue reads an object from the stream
	// v must be a settable value (v.CanSet() is true)
	DecodeValue(v reflect.Value) error
}
