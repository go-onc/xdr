// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package xdr

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.e43.eu/xdr/internal/errors"
)

func i32ptr(v int32) *int32 {
	return &v
}

func TestCodecsBasic(t *testing.T) {
	type nested struct {
		S    string `xdr:"maxlen:16"`
		Skip int32  `xdr:"-"`
		I    int32
	}

	type union1 struct {
		S         uint32    `xdr:"union:switch"`
		I         int32     `xdr:"union:0"`
		H         int64     `xdr:"union:1"`
		IP        *int32    `xdr:"union:2"`
		IPO       *int32    `xdr:"union:3/opt"`
		Str       string    `xdr:"union:4/maxlen:4"`
		StrFix    string    `xdr:"union:5/len:4"`
		Opaque    []byte    `xdr:"union:6/maxlen:4/opaque"`
		OpaqueFix [4]byte   `xdr:"union:7/opaque"`
		F         float32   `xdr:"union:8"`
		D         float64   `xdr:"union:9"`
		AI16      [4]uint16 `xdr:"union:10"`
		SI16      []uint16  `xdr:"union:11/maxlen:4"`
		AI8       [4]uint8  `xdr:"union:12"`
		SI8       []uint8   `xdr:"union:13/maxlen:4"`
		Skip      int32     `xdr:"-"`
		N         nested    `xdr:"union:14"`
		NP        *nested   `xdr:"union:15"`
		NPO       *nested   `xdr:"union:16/opt"`
		NA        [2]nested `xdr:"union:17"`
		NS        []nested  `xdr:"union:18/maxlen:4"`
	}

	testcases := []testcase{
		{
			Name:   "bool false",
			Object: false,
			Bytes:  []byte{0, 0, 0, 0},
		}, {
			Name:   "bool true",
			Object: true,
			Bytes:  []byte{0, 0, 0, 1},
		}, {
			Name:       "bool ???",
			Direction:  decodeTest,
			Object:     true,
			Bytes:      []byte{0, 0, 0, 2},
			DecErrorIs: errors.ErrInvalidValue,
		}, {
			Name:   "int32 -1",
			Object: int32(-1),
			Bytes:  []byte{0xff, 0xff, 0xff, 0xff},
		}, {
			Name:   "int32 0",
			Object: int32(0),
			Bytes:  []byte{0, 0, 0, 0},
		}, {
			Name:   "int32 1",
			Object: int32(1),
			Bytes:  []byte{0, 0, 0, 1},
		}, {
			Name: "Simple struct",
			Object: struct {
				X int32
				Y int64
			}{-1, 2},
			Bytes: []byte{0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 2},
		}, {
			Name:   "union1 I",
			Object: union1{S: 0, I: 0x12345678},
			Bytes:  []byte{0, 0, 0, 0, 0x12, 0x34, 0x56, 0x78},
		}, {
			Name:   "union1 H",
			Object: union1{S: 1, H: 0x12345678ABCDEFAB},
			Bytes:  []byte{0, 0, 0, 1, 0x12, 0x34, 0x56, 0x78, 0xAB, 0xCD, 0xEF, 0xAB},
		}, {
			Name:   "union1 IP (pointers are just ignored)",
			Object: union1{S: 2, IP: i32ptr(0x0EA7BEEF)},
			Bytes:  []byte{0, 0, 0, 2, 0x0E, 0xA7, 0xBE, 0xEF},
		}, {
			Name:   "union1 IPO nil",
			Object: union1{S: 3, IPO: nil},
			Bytes:  []byte{0, 0, 0, 3, 0, 0, 0, 0},
		}, {
			Name:   "union1 IPO !nil 0",
			Object: union1{S: 3, IPO: i32ptr(0)},
			Bytes:  []byte{0, 0, 0, 3, 0, 0, 0, 1, 0, 0, 0, 0},
		}, {
			Name:   "union1 IPO !nil val",
			Object: union1{S: 3, IPO: i32ptr(0x7EA0CAFE)},
			Bytes:  []byte{0, 0, 0, 3, 0, 0, 0, 1, 0x7E, 0xA0, 0xCA, 0xFE},
		}, {
			Name:       "union1 IPO multiple",
			Direction:  decodeTest,
			Object:     union1{},
			Bytes:      []byte{0, 0, 0, 3, 0, 0, 0, 2, 0x7E, 0xA0, 0xCA, 0xFE, 0x7E, 0xA0, 0xCA, 0xFE},
			DecErrorIs: errors.ErrInvalidValue,
		}, {
			Name:   "union1 Str 0",
			Object: union1{S: 4, Str: ""},
			Bytes:  []byte{0, 0, 0, 4, 0, 0, 0, 0},
		}, {
			Name:   "union1 Str 3",
			Object: union1{S: 4, Str: "Hi!"},
			Bytes:  []byte{0, 0, 0, 4, 0, 0, 0, 3, 'H', 'i', '!', 0},
		}, {
			Name:   "union1 Str 4",
			Object: union1{S: 4, Str: "Hi!!"},
			Bytes:  []byte{0, 0, 0, 4, 0, 0, 0, 4, 'H', 'i', '!', '!'},
		}, {
			Name:       "union1 Str 5",
			Object:     union1{S: 4, Str: "Hello"},
			Bytes:      []byte{0, 0, 0, 4, 0, 0, 0, 5, 'H', 'e', 'l', 'l', 'o', 0, 0, 0},
			EncErrorIs: errors.ErrLengthExceedsMax,
			DecErrorIs: errors.ErrLengthExceedsMax,
		}, {
			Name:   "union1 StrFix",
			Object: union1{S: 5, StrFix: "Hi!!"},
			Bytes:  []byte{0, 0, 0, 5, 'H', 'i', '!', '!'},
		}, {
			Name:       "union1 StrFix Wrong length",
			Direction:  encodeTest,
			Object:     union1{S: 5, StrFix: "Hello"},
			EncErrorIs: errors.ErrLengthIncorrect,
		}, {
			Name:   "union1 Opaque 0",
			Object: union1{S: 6, Opaque: nil},
			Bytes:  []byte{0, 0, 0, 6, 0, 0, 0, 0},
		}, {
			Name:   "union1 Opaque 1",
			Object: union1{S: 6, Opaque: []byte{0x0A}},
			Bytes:  []byte{0, 0, 0, 6, 0, 0, 0, 1, 0x0A, 0, 0, 0},
		}, {
			Name:   "union1 Opaque 4",
			Object: union1{S: 6, Opaque: []byte{0x0A, 0x0B, 0x0C, 0x0D}},
			Bytes:  []byte{0, 0, 0, 6, 0, 0, 0, 4, 0x0A, 0x0B, 0x0C, 0x0D},
		}, {
			Name:       "union1 Opaque 5",
			Object:     union1{S: 6, Opaque: []byte{0x0A, 0x0B, 0x0C, 0x0D, 0x0E}},
			Bytes:      []byte{0, 0, 0, 6, 0, 0, 0, 5, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0, 0, 0},
			EncErrorIs: errors.ErrLengthExceedsMax,
			DecErrorIs: errors.ErrLengthExceedsMax,
		}, {
			Name:   "union1 OpaqueFix",
			Object: union1{S: 7, OpaqueFix: [4]byte{0xF, 0xE, 0xD, 0xC}},
			Bytes:  []byte{0, 0, 0, 7, 0xF, 0xE, 0xD, 0xC},
		}, {
			Name:   "union1 F 0",
			Object: union1{S: 8, F: 0.0},
			Bytes:  []byte{0, 0, 0, 8, 0x00, 0x00, 0x00, 0x00},
		}, {
			Name:   "union1 F 1.0",
			Object: union1{S: 8, F: 1.0},
			Bytes:  []byte{0, 0, 0, 8, 0x3F, 0x80, 0x00, 0x00},
		},
		// Todo: These have multiple potential bit patterns, and may differ between
		//       systems. We should decide between canonicalisation (slower) or permitting
		//       this and removing (or relaxing) these tests
		{
			Name:   "union1 F +Inf",
			Object: union1{S: 8, F: float32(math.Inf(1))},
			Bytes:  []byte{0, 0, 0, 8, 0x7F, 0x80, 0x00, 0x00},
		}, {
			Name:   "union1 F -Inf",
			Object: union1{S: 8, F: float32(math.Inf(-1))},
			Bytes:  []byte{0, 0, 0, 8, 0xFF, 0x80, 0x00, 0x00},
		}, {
			Name:   "union1 F NaN",
			Object: union1{S: 8, F: float32(math.NaN())},
			Bytes:  []byte{0, 0, 0, 8, 0x7F, 0xC0, 0x00, 0x00},
			DecodeComparator: func(t *testing.T, xi, ai interface{}) {
				x, a := xi.(union1), ai.(union1)
				assert.Equal(t, math.IsNaN(float64(x.F)), math.IsNaN(float64(a.F)), "union1.F should be NaN")
				// Set both to zero to ignore them for subsequent step
				x.F, a.F = 0, 0
				assert.Equal(t, x, a, "Decoded objects should be equal")
			},
		},
		// End todo
		{
			Name:   "union1 D 0",
			Object: union1{S: 9, D: 0.0},
			Bytes:  []byte{0, 0, 0, 9, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		}, {
			Name:   "union1 D 1.0",
			Object: union1{S: 9, D: 1.0},
			Bytes:  []byte{0, 0, 0, 9, 0x3F, 0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		// See previous note about bit patterns/canonicalisation
		{
			Name:   "union1 D +Inf",
			Object: union1{S: 9, D: math.Inf(1)},
			Bytes:  []byte{0, 0, 0, 9, 0x7F, 0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		}, {
			Name:   "union1 D -Inf",
			Object: union1{S: 9, D: math.Inf(-1)},
			Bytes:  []byte{0, 0, 0, 9, 0xFF, 0xF0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		}, {
			Name:   "union1 D NaN",
			Object: union1{S: 9, D: math.NaN()},
			Bytes:  []byte{0, 0, 0, 9, 0x7F, 0xF8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DecodeComparator: func(t *testing.T, xi, ai interface{}) {
				x, a := xi.(union1), ai.(union1)
				assert.Equal(t, math.IsNaN(x.D), math.IsNaN(a.D), "union1.D should be NaN")
				// Set both to zero to ignore them for subsequent step
				x.D, a.D = 0, 0
				assert.Equal(t, x, a, "Decoded objects should be equal")
			},
		},
		// End todo
		// Encodings with per-element padding
		{
			Name:   "union1 AI16",
			Object: union1{S: 10, AI16: [4]uint16{0x1111, 0x2222, 0x3333, 0x4444}},
			Bytes: []byte{
				0, 0, 0, 10,
				// Padding is leading as these get widened to uint32s
				0x00, 0x00, 0x11, 0x11, 0x00, 0x00, 0x22, 0x22,
				0x00, 0x00, 0x33, 0x33, 0x00, 0x00, 0x44, 0x44,
			},
		}, {
			Name:   "union1 SI16 x2",
			Object: union1{S: 11, SI16: []uint16{0x1111, 0x2222}},
			Bytes: []byte{
				0, 0, 0, 11,
				0, 0, 0, 2, // Size counts elements
				0x00, 0x00, 0x11, 0x11, 0x00, 0x00, 0x22, 0x22,
			},
		}, {
			Name:   "union1 SI16 x4",
			Object: union1{S: 11, SI16: []uint16{0x1111, 0x2222, 0x3333, 0x4444}},
			Bytes: []byte{
				0, 0, 0, 11,
				0, 0, 0, 4,
				0x00, 0x00, 0x11, 0x11, 0x00, 0x00, 0x22, 0x22,
				0x00, 0x00, 0x33, 0x33, 0x00, 0x00, 0x44, 0x44,
			},
		}, {
			Name:   "union1 SI16 x5",
			Object: union1{S: 11, SI16: []uint16{0x1111, 0x2222, 0x3333, 0x4444, 0x5555}},
			Bytes: []byte{
				0, 0, 0, 11,
				0, 0, 0, 5,
				0x00, 0x00, 0x11, 0x11, 0x00, 0x00, 0x22, 0x22,
				0x00, 0x00, 0x33, 0x33, 0x00, 0x00, 0x44, 0x44,
				0x00, 0x00, 0x55, 0x55,
			},
			EncErrorIs: errors.ErrLengthExceedsMax,
			DecErrorIs: errors.ErrLengthExceedsMax,
		},
		// The following are very dumb encodings of byte slices, but we verify that they
		// work. This is basically the same case as the previous ones with padding, but
		// verifying that uint8 slices don't accidentally end up as opaque
		{
			Name:   "union1 AI8",
			Object: union1{S: 12, AI8: [4]uint8{0x11, 0x22, 0x33, 0x44}},
			Bytes: []byte{
				0, 0, 0, 12,
				0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x22,
				0x00, 0x00, 0x00, 0x33, 0x00, 0x00, 0x00, 0x44,
			},
		}, {
			Name:   "union1 SI8 x2",
			Object: union1{S: 13, SI8: []uint8{0x11, 0x22}},
			Bytes: []byte{
				0, 0, 0, 13,
				0, 0, 0, 2, // Size counts elements
				0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x22,
			},
		}, {
			Name:   "union1 SI8 x4",
			Object: union1{S: 13, SI8: []uint8{0x11, 0x22, 0x33, 0x44}},
			Bytes: []byte{
				0, 0, 0, 13,
				0, 0, 0, 4,
				0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x22,
				0x00, 0x00, 0x00, 0x33, 0x00, 0x00, 0x00, 0x44,
			},
		}, {
			Name:   "union1 SI8 x5",
			Object: union1{S: 13, SI8: []uint8{0x11, 0x22, 0x33, 0x44, 0x55}},
			Bytes: []byte{
				0, 0, 0, 13,
				0, 0, 0, 5,
				0x00, 0x00, 0x00, 0x11, 0x00, 0x00, 0x00, 0x22,
				0x00, 0x00, 0x00, 0x33, 0x00, 0x00, 0x00, 0x44,
				0x00, 0x00, 0x00, 0x55,
			},
			EncErrorIs: errors.ErrLengthExceedsMax,
			DecErrorIs: errors.ErrLengthExceedsMax,
		},
		// Nested structures, combined with other options
		{
			Name:   "union1 N",
			Object: union1{S: 14, N: nested{S: "hi", I: 0x12345678}},
			Bytes: []byte{
				0, 0, 0, 14,
				0, 0, 0, 2, 'h', 'i', 0, 0,
				0x12, 0x34, 0x56, 0x78,
			},
		}, {
			Name:   "union1 NP",
			Object: union1{S: 15, NP: &nested{S: "hi", I: 0x12345678}},
			Bytes: []byte{
				0, 0, 0, 15,
				0, 0, 0, 2, 'h', 'i', 0, 0,
				0x12, 0x34, 0x56, 0x78,
			},
		}, {
			Name:   "union1 NPO nil",
			Object: union1{S: 16, NPO: nil},
			Bytes: []byte{
				0, 0, 0, 16,
				0, 0, 0, 0,
			},
		}, {
			Name:   "union1 NPO not-nil",
			Object: union1{S: 16, NPO: &nested{S: "hi", I: 0x12345678}},
			Bytes: []byte{
				0, 0, 0, 16,
				0, 0, 0, 1,
				0, 0, 0, 2, 'h', 'i', 0, 0,
				0x12, 0x34, 0x56, 0x78,
			},
		}, {
			Name: "union1 NA",
			Object: union1{
				S: 17,
				NA: [2]nested{
					{S: "hi", I: 0x12345678},
					{S: "longer string", I: 0xC0DEC},
				},
			},
			Bytes: []byte{
				0, 0, 0, 17,
				0, 0, 0, 2, 'h', 'i', 0, 0, 0x12, 0x34, 0x56, 0x78,
				0, 0, 0, 13, 'l', 'o', 'n', 'g', 'e', 'r', ' ', 's', 't', 'r', 'i', 'n', 'g', 0, 0, 0, 0x00, 0x0C, 0x0D, 0xEC,
			},
		}, {
			Name: "union1 NS empty",
			Object: union1{
				S:  18,
				NS: nil,
			},
			Bytes: []byte{
				0, 0, 0, 18,
				0, 0, 0, 0,
			},
		}, {
			Name: "union1 NS 1",
			Object: union1{
				S: 18,
				NS: []nested{
					{S: "hi", I: 0x12345678},
				},
			},
			Bytes: []byte{
				0, 0, 0, 18,
				0, 0, 0, 1,
				0, 0, 0, 2, 'h', 'i', 0, 0, 0x12, 0x34, 0x56, 0x78,
			},
		}, {
			Name: "union1 NS 2",
			Object: union1{
				S: 18,
				NS: []nested{
					{S: "hi", I: 0x12345678},
					{S: "longer string", I: 0xC0DEC},
				},
			},
			Bytes: []byte{
				0, 0, 0, 18,
				0, 0, 0, 2,
				0, 0, 0, 2, 'h', 'i', 0, 0, 0x12, 0x34, 0x56, 0x78,
				0, 0, 0, 13, 'l', 'o', 'n', 'g', 'e', 'r', ' ', 's', 't', 'r', 'i', 'n', 'g', 0, 0, 0, 0x00, 0x0C, 0x0D, 0xEC,
			},
		}, {
			Name: "union1 NS 4",
			Object: union1{
				S: 18,
				NS: []nested{
					{S: "hi", I: 0x12345678},
					{S: "longer string", I: 0xC0DEC},
					{S: "0123456789abcdef", I: 0x12345678},
					{S: "", I: 0},
				},
			},
			Bytes: []byte{
				0, 0, 0, 18,
				0, 0, 0, 4,
				0, 0, 0, 2, 'h', 'i', 0, 0, 0x12, 0x34, 0x56, 0x78,
				0, 0, 0, 13, 'l', 'o', 'n', 'g', 'e', 'r', ' ', 's', 't', 'r', 'i', 'n', 'g', 0, 0, 0, 0x00, 0x0C, 0x0D, 0xEC,
				0, 0, 0, 16, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f', 0x12, 0x34, 0x56, 0x78,
				0, 0, 0, 0, 0, 0, 0, 0,
			},
		}, {
			Name: "union1 NS 4",
			Object: union1{
				S: 18,
				NS: []nested{
					{S: "hi", I: 0x12345678},
					{S: "longer string", I: 0xC0DEC},
					{S: "0123456789abcdef", I: 0x12345678},
					{S: "", I: 0},
					{S: "", I: 0x11111111},
				},
			},
			Bytes: []byte{
				0, 0, 0, 18,
				0, 0, 0, 5,
				0, 0, 0, 2, 'h', 'i', 0, 0, 0x12, 0x34, 0x56, 0x78,
				0, 0, 0, 13, 'l', 'o', 'n', 'g', 'e', 'r', ' ', 's', 't', 'r', 'i', 'n', 'g', 0, 0, 0, 0x00, 0x0C, 0x0D, 0xEC,
				0, 0, 0, 16, '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f', 0x12, 0x34, 0x56, 0x78,
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0x11, 0x11, 0x11, 0x11,
			},
			EncErrorIs: errors.ErrLengthExceedsMax,
			DecErrorIs: errors.ErrLengthExceedsMax,
		},
		// Tests skipped on 64-bit platforms
		// These check behaviour on systems where the sizes encoded in the XDR may exceed
		// the maximum value of an int()
		{
			Name:       "[32-bit only] MaxLen opaque",
			ShouldSkip: skipOn64,
			Direction:  decodeTest,
			Object: struct {
				Opaque []byte `xdr:"opaque"` // No max length specified
			}{},
			ReaderFactory: infintelyPaddedReaderFactory([]byte{
				0x80, 0x00, 0x00, 0x00,
			}),
			DecErrorIs: errors.ErrLengthExceedsPlatformLimit,
		}, {
			Name:       "[32-bit only] MaxU32 opaque",
			ShouldSkip: skipOn64,
			Direction:  decodeTest,
			Object: struct {
				Opaque []byte `xdr:"opaque"` // No max length specified
			}{},
			ReaderFactory: infintelyPaddedReaderFactory([]byte{
				0xFF, 0xFF, 0xFF, 0xFF,
			}),
			DecErrorIs: errors.ErrLengthExceedsPlatformLimit,
		}, {
			Name:       "[32-bit only] MaxLen []int32",
			ShouldSkip: skipOn64,
			Direction:  decodeTest,
			Object: struct {
				Blob []int32 // No max length specified
			}{},
			ReaderFactory: infintelyPaddedReaderFactory([]byte{
				0x80, 0x00, 0x00, 0x00,
			}),
			DecErrorIs: errors.ErrLengthExceedsPlatformLimit,
		}, {
			Name:       "[32-bit only] MaxLen fixed string",
			ShouldSkip: skipOn64,
			Direction:  decodeTest,
			Object: struct {
				Blob string `xdr:"len:0x80000000"`
			}{},
			ReaderFactory: infintelyPaddedReaderFactory(nil),
			DecErrorIs:    errors.ErrLengthExceedsPlatformLimit,
		}, {
			Name:       "[32-bit only] MaxLen string",
			ShouldSkip: skipOn64,
			Direction:  decodeTest,
			Object: struct {
				Blob string // No max length specified
			}{},
			ReaderFactory: infintelyPaddedReaderFactory([]byte{
				0x80, 0x00, 0x00, 0x00,
			}),
			DecErrorIs: errors.ErrLengthExceedsPlatformLimit,
		},
	}

	RunTestcases(t, testcases)
}
