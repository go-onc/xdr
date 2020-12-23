// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"io"
	"io/ioutil"
)

type opaqueReader struct {
	lr     io.LimitedReader
	padLen byte
}

func newOpaqueReader(r io.Reader, len int64) *opaqueReader {
	return &opaqueReader{
		lr: io.LimitedReader{
			R: r,
			N: len,
		},
		padLen: uint8(((len + 3) & ^3) - len),
	}
}

func (o *opaqueReader) Read(p []byte) (int, error) {
	return o.lr.Read(p)
}

func (o *opaqueReader) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, &o.lr)
}

func (o *opaqueReader) Close() error {
	o.lr.N += int64(o.padLen)
	_, err := io.Copy(ioutil.Discard, &o.lr)
	return err
}

var _ io.Reader = &opaqueReader{}
var _ io.ReadCloser = &opaqueReader{}
var _ io.WriterTo = &opaqueReader{}
