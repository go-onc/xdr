// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package xdr

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io/ioutil"
	"testing"
)

func EncodeBenchmarkCommon(b *testing.B, ob interface{}) {
	b.Run("XDRMarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := Marshal(ob)
			if err != nil {
				b.Fatalf("Marshal: %s", err)
			}
		}
	})

	b.Run("JSONMarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(ob)
			if err != nil {
				b.Fatalf("json.Marshal: %s", err)
			}
		}
	})

	b.Run("XDRWriteDiscard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := Write(ioutil.Discard, ob)
			if err != nil {
				b.Fatalf("Write: %s", err)
			}
		}
	})

	b.Run("XDREncoderDiscard", func(b *testing.B) {
		w := NewEncoder(ioutil.Discard)
		for i := 0; i < b.N; i++ {
			err := w.Encode(ob)
			if err != nil {
				b.Fatalf("Encode: %s", err)
			}
		}
	})

	b.Run("GobEncoderDiscard", func(b *testing.B) {
		w := gob.NewEncoder(ioutil.Discard)
		for i := 0; i < b.N; i++ {
			err := w.Encode(ob)
			if err != nil {
				b.Fatalf("Encode: %s", err)
			}
		}
	})

	b.Run("JSONEncoderDiscard", func(b *testing.B) {
		w := json.NewEncoder(ioutil.Discard)
		for i := 0; i < b.N; i++ {
			err := w.Encode(ob)
			if err != nil {
				b.Fatalf("Encode: %s", err)
			}
		}
	})

	b.Run("XDREncoderBuffer", func(b *testing.B) {
		var buf bytes.Buffer
		w := NewEncoder(&buf)
		for i := 0; i < b.N; i++ {
			err := w.Encode(ob)
			if err != nil {
				b.Fatalf("Encode: %s", err)
			}

			if (i % 2048) == 0 {
				buf.Reset()
			}
		}
	})

	b.Run("GobEncoderBuffer", func(b *testing.B) {
		var buf bytes.Buffer
		w := gob.NewEncoder(&buf)
		for i := 0; i < b.N; i++ {
			err := w.Encode(ob)
			if err != nil {
				b.Fatalf("Encode: %s", err)
			}

			if (i % 2048) == 0 {
				buf.Reset()
			}
		}
	})

	b.Run("JSONEncoderBuffer", func(b *testing.B) {
		var buf bytes.Buffer
		w := json.NewEncoder(&buf)
		for i := 0; i < b.N; i++ {
			err := w.Encode(ob)
			if err != nil {
				b.Fatalf("Encode: %s", err)
			}

			if (i % 2048) == 0 {
				buf.Reset()
			}
		}
	})
}

func BenchmarkInt32Encode(b *testing.B) {
	EncodeBenchmarkCommon(b, int32(123))
}

func BenchmarkInt64Encode(b *testing.B) {
	EncodeBenchmarkCommon(b, int64(768))
}

func BenchmarkStringEncode(b *testing.B) {
	EncodeBenchmarkCommon(b, "Hello World")
}

func BenchmarkSimpleStructEncode(b *testing.B) {
	type S struct {
		X int32
		Y int64
		S string `xdr:"maxlen:32"`
		O []byte `xdr:"maxlen:32/opaque"`
		// IP1 *int32 `xdr:"opt" json:",omitempty"`
		// IP2 *int32 `xdr:"opt" json:",omitempty"`
	}

	s := &S{
		X: 123456,
		Y: 12345678,
		S: "Hello Encoders",
		O: []byte("Byte Slice"),
		// IP1: new(int32),
		// IP2: nil,
	}

	EncodeBenchmarkCommon(b, s)
}

func BenchmarkUnionStructsEncode(b *testing.B) {
	type S1 struct {
		Frob int32
		Glob int32
	}

	type S2 struct {
		Foo int32
		Bar string `xdr:"maxlen:32"`
	}

	type S3 struct {
		Foo *S1 `xdr:"opt" json:"foo,omitempty"`
		Baz int32
	}

	type U struct {
		Switch uint32 `xdr:"union:switch"`
		S1     *S1    `xdr:"union:0" json:"s1,omitempty"`
		S2     *S2    `xdr:"union:1" json:"s2,omitempty"`
		S3     *S3    `xdr:"union:2" json:"s3,omitempty"`
	}

	vals := []U{
		{Switch: 0, S1: &S1{123, 456}},
		{Switch: 1, S2: &S2{789, "A string"}},
		{Switch: 2, S3: &S3{&S1{65535, 1024}, 512}},
		{Switch: 1, S2: &S2{789, "A second string"}},
		{Switch: 2, S3: &S3{nil, 256}},
	}
	EncodeBenchmarkCommon(b, vals)
}

// These aren't optimal but it's somewhat tricky to do much better
// with the interface the JSON package offers
type complex64t complex64

func (c complex64t) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float32{real(c), imag(c)})
}

type complex128t complex128

func (c complex128t) MarshalJSON() ([]byte, error) {
	return json.Marshal([]float64{real(c), imag(c)})
}

func BenchmarkUnionComplex(b *testing.B) {
	type U struct {
		Switch uint32       `xdr:"union:switch"`
		C64    *complex64t  `xdr:"union:3" json:"c64,omitempty"`
		C128   *complex128t `xdr:"union:4" json:"c128,omitempty"`
	}

	c64 := complex64t(complex(float32(1.0), float32(2.0)))
	c128 := complex128t(complex(float64(2.0), float64(4.0)))

	vals := []U{
		{Switch: 3, C64: &c64},
		{Switch: 4, C128: &c128},
	}
	EncodeBenchmarkCommon(b, vals)

}
