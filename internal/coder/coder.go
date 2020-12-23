// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sync"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
	"go.e43.eu/xdr/internal/tags"
)

const (
	// maxUint is the maximum value a uint can hold
	maxUint = ^uint(0)
	// maxInt is the maximum value an int can hold
	maxInt = int(maxUint >> 1)
)

var (
	marshalerType = reflect.TypeOf((*xdrinterfaces.Marshaler)(nil)).Elem()
)

type xType struct {
	Type       reflect.Type
	EncodedTag string
}

type Coder struct {
	knownBaseCodecs sync.Map // map[reflect.Type]xCodec
	knownCodecs     sync.Map // map[xType]xCodec
}

func NewCoder() *Coder {
	return new(Coder)
}

func (cr *Coder) getBaseCodec(t reflect.Type) xCodec {
	c, ok := cr.knownBaseCodecs.Load(t)
	if ok {
		return c.(xCodec)
	}

	// Less common case: need to construct a codec
	return cr.getNewCodec(xType{t, ""}, nil)
}

func (cr *Coder) getCodec(t reflect.Type, tag tags.XDRTag) xCodec {
	// Common case: already known; just lookup type
	xt := xType{t, tag.ByteString()}
	c, ok := cr.knownCodecs.Load(xt)
	if ok {
		return c.(xCodec)
	}

	// Less common case: need to construct a codec
	return cr.getNewCodec(xt, tag)
}

// Types of object you are prevented from registering codecs for
var prohibitedCustomCodecKinds = map[reflect.Kind]struct{}{
	reflect.Invalid: struct{}{},

	// Prohibited because these would interact poorly with tagged fields in structs.
	// These problems are not unsolvable, but we are protecting against them for now
	reflect.Array:  struct{}{},
	reflect.Slice:  struct{}{},
	reflect.String: struct{}{},
	reflect.Map:    struct{}{},

	// Would make behaviour of pointers in general inconsistent
	// It wouldn't be difficult to support this with good reason, however.
	reflect.Ptr: struct{}{},

	// These make little sense to support
	reflect.Chan: struct{}{},
	reflect.Func: struct{}{},

	reflect.UnsafePointer: struct{}{},
}

// These are blocked because implementing different behaviour for
// the primitive types would be incredibly confusing
var prohibitedPrimitives = map[reflect.Type]struct{}{
	reflect.TypeOf(false):         struct{}{},
	reflect.TypeOf(int8(0)):       struct{}{},
	reflect.TypeOf(int16(0)):      struct{}{},
	reflect.TypeOf(int32(0)):      struct{}{},
	reflect.TypeOf(int64(0)):      struct{}{},
	reflect.TypeOf(int(0)):        struct{}{},
	reflect.TypeOf(uint8(0)):      struct{}{},
	reflect.TypeOf(uint16(0)):     struct{}{},
	reflect.TypeOf(uint32(0)):     struct{}{},
	reflect.TypeOf(uint64(0)):     struct{}{},
	reflect.TypeOf(uint(0)):       struct{}{},
	reflect.TypeOf(uintptr(0)):    struct{}{},
	reflect.TypeOf(float32(0)):    struct{}{},
	reflect.TypeOf(float64(0)):    struct{}{},
	reflect.TypeOf(complex64(0)):  struct{}{},
	reflect.TypeOf(complex128(0)): struct{}{},
}

func (cr *Coder) RegisterCodec(template interface{}, c xdrinterfaces.Codec) {
	cr.RegisterCodecReflect(reflect.TypeOf(template), c)
}

func (cr *Coder) RegisterCodecReflect(t reflect.Type, c xdrinterfaces.Codec) {
	if _, badKind := prohibitedCustomCodecKinds[t.Kind()]; badKind {
		panic(fmt.Sprintf("Attempt to register codec for type %s which is of a prohibited kind", t))
	}

	if _, isPrimitive := prohibitedPrimitives[t]; isPrimitive {
		panic(fmt.Sprintf("Attempt to register codec for primitive %s is prohibited", t))
	}

	xt := xType{t, ""}
	existing, found := cr.knownCodecs.LoadOrStore(xt, c)
	if found && toOriginalCodec(existing.(xCodec)) != c {
		panic(fmt.Sprintf("Attempt to register codec '%s' for type '%s' but '%s' is already registered", c, t, existing))
	}
}

func (cr *Coder) getNewCodec(xt xType, tag tags.XDRTag) xCodec {
	// We create a "deferred codec" in order to handle cycles in the type graph. Note
	// that we also need to be prepared for the possibility that another goroutine
	// is constructing a type related to this one or looking this one up simultaneously,
	// so this codec must not explode if called while being constructed
	//
	// Every call to the deferred codec will block until we finish constructing the
	// real one.
	dc := newDeferredCodec()

	// We were potentially racing against someone else to build the codec up to this point,
	// so we must check that here. If someone else has built (or is building) the codec,
	// we'll go with theirs instead
	c, ok := cr.knownCodecs.LoadOrStore(xt, dc)
	if ok {
		return c.(xCodec)
	}

	// Actually construct the codec
	cc := toXCodec(cr.buildCodec(xt.Type, tag), xt.Type)

	// Publish our newly built: Replace the deferred one in the store, and close the signalling channel
	// so that anyone waiting on us may progress
	cr.knownCodecs.Store(xt, cc)
	if tag.Empty() {
		cr.knownBaseCodecs.Store(xt.Type, cc)
	}
	dc.resolve(cc)
	return cc
}

func (cr *Coder) buildCodec(t reflect.Type, tag tags.XDRTag) xdrinterfaces.Codec {
	// Handle certain special case tags first
	switch tag.Kind() {
	case tags.Opt:
		// Opt can be applied generically to a number of different types, so
		// start with that
		return makeOptCodec(cr, t, tag)
	}

	k := t.Kind()

	// Delegate straight through to types with their own tag handling
	switch k {
	case reflect.Ptr:
		return makePtrCodec(cr, t, tag)

	case reflect.String:
		return makeStringCodec(t, tag)

	case reflect.Array:
		return makeArrayCodec(cr, t, tag)

	case reflect.Slice:
		return makeSliceCodec(cr, t, tag)

	case reflect.Map:
		return makeMapCodec(cr, t, tag)
	}

	// None of the remaining types admit any tags
	if !tag.Empty() {
		return &errorCodec{errors.InvalidTagForTypeError{t, tag}}
	}

	switch {
	case t.Implements(marshalerType):
		return &marshalerCodecI
	}

	switch k {
	case reflect.Bool:
		return boolCodecI
	case reflect.Int8:
		return int8CodecI
	case reflect.Int16:
		return int16CodecI
	case reflect.Int32:
		return int32CodecI
	case reflect.Uint8:
		return uint8CodecI
	case reflect.Uint16:
		return uint16CodecI
	case reflect.Uint32:
		return uint32CodecI
	case reflect.Int64:
		return hyperCodecI
	case reflect.Uint64:
		return uhyperCodecI
	case reflect.Float32:
		return floatCodecI
	case reflect.Float64:
		return doubleCodecI
	case reflect.Complex64:
		return complex64CodecI
	case reflect.Complex128:
		return complex128CodecI
	case reflect.Struct:
		return makeStructCodec(cr, t)
	default:
		return &errorCodec{errors.InvalidTypeError{t}}
	}
}

func (cr *Coder) NewEncoder(w io.Writer) xdrinterfaces.Encoder {
	return cr.newEncoder(w)
}

func (cr *Coder) newEncoder(w io.Writer) *encoder {
	e := encoderPool.Get().(*encoder)
	e.reset(cr, w)
	return e
}

func (cr *Coder) NewDecoder(r io.Reader) xdrinterfaces.Decoder {
	return cr.newDecoder(r)
}

func (cr *Coder) newDecoder(r io.Reader) *decoder {
	d := decoderPool.Get().(*decoder)
	d.r = r
	d.cr = cr
	return d
}

func (cr *Coder) Marshal(o interface{}) ([]byte, error) {
	e := marshalEncoderPool.Get().(*marshalEncoder)
	defer e.release()

	e.reset(cr)
	err := e.Encode(o)

	return append([]byte(nil), e.b.Bytes()...), err
}

func (cr *Coder) Unmarshal(buf []byte, op interface{}) error {
	var r bytes.Reader
	r.Reset(buf)
	d := cr.newDecoder(&r)
	err := d.Decode(op)
	d.release()
	return err
}

var writerPool = sync.Pool{
	New: func() interface{} {
		return bufio.NewWriter(nil)
	},
}

func (cr *Coder) Write(w io.Writer, o interface{}) error {
	switch w.(type) {
	case *bytes.Buffer, *bufio.Writer:
		// Already buffered
		e := cr.newEncoder(w)
		err := e.Encode(o)
		e.release()
		return err
	}

	bw := writerPool.Get().(*bufio.Writer)
	bw.Reset(w)
	e := cr.newEncoder(bw)
	err := e.Encode(o)
	e.release()
	if err == nil {
		err = bw.Flush()
	}
	bw.Reset(nil)
	writerPool.Put(bw)
	return err
}

func (cr *Coder) Read(r io.Reader, op interface{}) error {
	d := cr.newDecoder(r)
	err := d.Decode(op)
	d.release()
	return err
}
