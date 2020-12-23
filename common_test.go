// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package xdr

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"io/ioutil"
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testDirection int

const (
	bothTest testDirection = iota
	encodeTest
	decodeTest
)

// comparingWriter is an io.Writer which immediately compares every byte
// written to it against the values read from the passed reader. This
// enables capturing the call stack at the time any discrepancy in the
// written data occurs
//
// It captures the written data so that a final comparison (which may somtimes
// be more informative) can also be made
type comparingWriter struct {
	T *testing.T

	// The reader
	R io.Reader

	// Error returned by reader
	Rerr error

	// Bytes written
	B []byte

	// Bytes expected
	X []byte
}

func newComparingWriter(t *testing.T, r io.Reader) *comparingWriter {
	return &comparingWriter{
		T: t,
		R: r,
	}
}

func (w *comparingWriter) Write(buf []byte) (int, error) {
	w.T.Helper()

	w.B = append(w.B, buf...)

	// Gather the expected bytes
	var expected []byte
	if w.Rerr == nil {
		expected = make([]byte, len(buf))
		nr, err := io.ReadFull(w.R, expected)
		expected = expected[0:nr]
		w.X = append(w.X, expected...)
		if err == io.ErrUnexpectedEOF {
			err = io.EOF
		}

		if err != nil {
			require.Equal(w.T, io.EOF, err, "comparingWriter: Comparison reader returned non-EOF error")
			assert.Failf(w.T, "Attempt to write after end", "Attempt to write %d bytes after end of expected data", len(buf)-nr)
			w.Rerr = err
		}
	}

	// If we read any bytes, cross compare them
	if len(expected) != 0 {
		assert.Equalf(w.T, expected, buf[0:len(expected)], "Expected equal value during %d byte write", len(buf))
	}

	return len(buf), nil
}

func (w *comparingWriter) Assert() {
	buf := make([]byte, 1024)
	err := w.Rerr

	var n int
	for err == nil {
		n, err = w.R.Read(buf)
		w.X = append(w.X, buf[0:n]...)
		require.Equal(w.T, io.EOF, err, "comparingWriter: Comparison reader must only return io.EOF error")
	}

	assert.Equalf(w.T, w.X, w.B, "Expected written data to match expected")
}

// singleByteReader is a really annoying io.Reader which returns a single byte at a time
type singleByteReader struct {
	R io.Reader
}

func (r *singleByteReader) Read(buf []byte) (int, error) {
	switch {
	case len(buf) == 0:
		return 0, nil
	default:
		return r.R.Read(buf[0:1])
	}
}

// Returns a reader with a prefix giveb by buf and which never stops after that
func infintelyPaddedReaderFactory(buf []byte) func(*testing.T, testDirection) io.Reader {
	return func(*testing.T, testDirection) io.Reader {
		return io.MultiReader(bytes.NewBuffer(buf), rand.Reader)
	}
}

const (
	// maxUint is the maximum value a uint can hold
	maxUint = ^uint(0)
	// maxInt is the maximum value an int can hold
	maxInt = int(maxUint >> 1)
)

// Skips this test if on a 64-bit platform
func skipOn64(_ *testing.T, _ testDirection) (bool, string) {
	return int64(maxInt) == int64(math.MaxInt64), "Test not valid on 64-bit platforms"
}

// Skips this test if on a 32-bit platform
func skipOn32(_ *testing.T, _ testDirection) (bool, string) {
	return int64(maxInt) == int64(math.MaxInt32), "Test not valid on 32-bit platforms"
}

type testcase struct {
	// Name of this test case
	Name string

	// Which directions to run this test in (defaults to both)
	Direction testDirection

	// When to skip this test, for tests which should only run sometimes
	// (e.g. tests which only work on 32-bit or 64-bit platforms)
	//
	// If returning true, it's a good idea to give a reason
	ShouldSkip func(*testing.T, testDirection) (bool, string)

	// The object to marshal, or to use for comparison on unmarshalling
	Object interface{}

	// The encoded representation of the object
	Bytes []byte

	// Returns a reader which returns a representation of the object. If specified,
	// will be used instead of Bytes
	//
	// Use testDirection to tell if you're in the encoder/decoder test, in case you've
	// made an asshole reader for testing the decoder and wish to inhibit its use on
	// the decoder side
	ReaderFactory func(*testing.T, testDirection) io.Reader

	// Error expected on en/decode
	EncErrorIs error
	DecErrorIs error

	// Comparator to use (instead of default) after successful decoding
	// The NaN tests use this because NaN != NaN, so normal comparisons won't work
	DecodeComparator func(t *testing.T, expt, actual interface{})
}

func RunTestcases(t *testing.T, tcs []testcase) {
	// Preprocess testcases:
	// * Add ReaderFactories for those which specified Bytes
	// * Insert the defualt DecoderComparator
	for i := range tcs {
		tc := &tcs[i]

		if tc.ReaderFactory == nil {
			tc.ReaderFactory = func(*testing.T, testDirection) io.Reader {
				return bytes.NewBuffer(tc.Bytes)
			}
		}

		if tc.DecodeComparator == nil {
			tc.DecodeComparator = func(t *testing.T, l, r interface{}) {
				t.Helper()
				assert.Equal(t, l, r, "unmarshal output should match")
			}
		}

		if tc.ShouldSkip == nil {
			tc.ShouldSkip = func(*testing.T, testDirection) (bool, string) {
				return false, ""
			}
		}
	}

	generatedTestcases := append([]testcase(nil), tcs...)
	t.Parallel()

	// For every case where the decoder is tested, build a variant with
	// the single byte reader
	for _, tc := range tcs {
		if tc.Direction == encodeTest {
			continue
		}
		tc := tc
		tc.Name += "+singleByteReader"
		tc.Direction = decodeTest
		innerFactory := tc.ReaderFactory
		tc.ReaderFactory = func(t *testing.T, d testDirection) io.Reader {
			return &singleByteReader{innerFactory(t, d)}
		}

		generatedTestcases = append(generatedTestcases, tc)
	}

	for _, tc := range generatedTestcases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			if tc.Direction != decodeTest {
				t.Run("Encode", func(t *testing.T) {
					t.Parallel()
					if skip, reason := tc.ShouldSkip(t, encodeTest); skip {
						t.Skip(reason)
					}

					var w io.Writer
					if tc.EncErrorIs != nil {
						w = ioutil.Discard
					} else {
						w = newComparingWriter(t, tc.ReaderFactory(t, encodeTest))
					}
					e := NewEncoder(w)
					err := e.Encode(tc.Object)
					if tc.EncErrorIs != nil {
						require.Error(t, err, "Encoding should have returned an error")
						require.Truef(t, errors.Is(err, tc.EncErrorIs), "Error expected to be %s, but was %s", tc.EncErrorIs, err)
					} else {
						require.NoError(t, err, "Encode should succeed")
						w.(*comparingWriter).Assert()
					}
				})

				// We have optimisatiosn which only kick in when you pass a pointer,
				// so run a variant with a pointer-to-object
				t.Run("EncodePtr", func(t *testing.T) {
					t.Parallel()
					if skip, reason := tc.ShouldSkip(t, encodeTest); skip {
						t.Skip(reason)
					}

					var w io.Writer
					if tc.EncErrorIs != nil {
						w = ioutil.Discard
					} else {
						w = newComparingWriter(t, tc.ReaderFactory(t, encodeTest))
					}
					e := NewEncoder(w)
					v := reflect.ValueOf(tc.Object)
					vp := reflect.New(v.Type())
					vp.Elem().Set(v)
					err := e.Encode(vp.Interface())
					if tc.EncErrorIs != nil {
						require.Error(t, err, "Encoding should have returned an error")
						require.Truef(t, errors.Is(err, tc.EncErrorIs), "Error expected to be %s, but was %s", tc.EncErrorIs, err)
					} else {
						require.NoError(t, err, "Encode should succeed")
						w.(*comparingWriter).Assert()
					}
				})
			}

			if tc.Direction != encodeTest {
				t.Run("Decode", func(t *testing.T) {
					t.Parallel()
					if skip, reason := tc.ShouldSkip(t, decodeTest); skip {
						t.Skip(reason)
					}

					r := tc.ReaderFactory(t, decodeTest)
					d := NewDecoder(r)

					// If tc.Object is of type T, then construct new(T)
					tgtp := reflect.New(reflect.TypeOf(tc.Object)).Interface()

					// Do the read
					err := d.Decode(tgtp)
					if tc.DecErrorIs != nil {
						if assert.Error(t, err, "Decoding should have returned an error") {
							assert.Truef(t, errors.Is(err, tc.DecErrorIs), "Error expected to be %s, but was %s", tc.DecErrorIs, err)
						} else {
							t.Logf("Returned %+v", tgtp)
						}
					} else {
						require.NoError(t, err, "Decode should succeed")
						// Assert that we drained the reader
						var trail bytes.Buffer
						nb, err := io.Copy(&trail, r)
						assert.NoError(t, err, "Should have no error draining tail")
						assert.Equalf(t, int64(0), nb, "Decoder left trailing bytes after end: %x", trail.Bytes())

						// Dereference the pointer to get a T for comparison purposes
						o := reflect.ValueOf(tgtp).Elem().Interface()
						tc.DecodeComparator(t, o, tc.Object)
					}
				})
			}
		})
	}
}
