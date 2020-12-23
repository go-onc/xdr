// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package errors

import (
	"fmt"
	"reflect"
	"strings"

	"go.e43.eu/xdr/internal/tags"
)

const (
	// maxUint is the maximum value a uint can hold
	maxUint = ^uint(0)
	// maxInt is the maximum value an int can hold
	maxInt = int(maxUint >> 1)
)

type xerror string

func (e xerror) Error() string {
	return string(e)
}

const (
	// Array or slice longer than permitted by the schema
	// (or XDR; for values where the schema specifies no limit, it is implicitly
	// treated as if 0xFFFFFFFF were specified; though in that case the error can
	// only be reached on encode)
	ErrLengthExceedsMax = xerror("xdr: Variable length object too long")

	// Array or slice length longer than we can decode
	//
	// This error means that a received length was larger than can be represented
	// as the Go `int` type but less than any maximum specified by the schema.
	//
	// This can only occur on 32-bit platforms, if the schema permits arrays of
	// more than 0x8000_0000 items. It can never occur on 64-bit platforms, as
	// XDR does not allow arrays of more than 2^32-1 items.
	ErrLengthExceedsPlatformLimit = xerror("xdr: Variable length object too long for platform")

	// Length of fixed length object incorrect
	//
	// This is returned when attempting to marshal an object of the wrong length
	ErrLengthIncorrect = xerror("xdr: Length incorrect")

	// Union switch arm undefined
	ErrUnionSwitchArmUndefined = xerror("xdr: Union switch arm undefined")

	// ReadObject expected pointer parameter
	ErrNotPointer = xerror("xdr: Expected pointer parameter")

	// Invalid value for type
	ErrInvalidValue = xerror("xdr: Invalid value for type")

	// Pointer was unexpectedly nil
	ErrNilPointer = xerror("xdr: Unexpected nil pointer")
)

type InvalidTypeError struct {
	T reflect.Type
}

func (e InvalidTypeError) Error() string {
	return fmt.Sprintf("xdr: Type '%s' unsupported", e.T)
}

type InvalidTagForTypeError struct {
	T   reflect.Type
	Tag tags.XDRTag
}

func (e InvalidTagForTypeError) Error() string {
	return fmt.Sprintf("xdr: Tag '%s' unsupported for type '%s'", e.Tag, e.T)
}

type LengthError struct {
	Actual, Max uint64
}

func (err LengthError) Is(target error) bool {
	switch target {
	case ErrLengthExceedsMax:
		return err.Actual > err.Max
	case ErrLengthExceedsPlatformLimit:
		return err.Actual > uint64(maxInt)
	default:
		return false
	}
}

func (err LengthError) Error() string {
	if err.Actual > err.Max {
		return fmt.Sprintf("%s (%d > %d)", ErrLengthExceedsMax, err.Actual, err.Max)
	} else {
		return fmt.Sprintf("%s (%d > %d)", ErrLengthExceedsPlatformLimit, err.Actual, maxInt)
	}
}

type FieldError struct {
	Underlying error
	Path       string
}

func (err FieldError) Unwrap() error {
	return err.Underlying
}

func (err FieldError) Error() string {
	uerr := strings.TrimPrefix(err.Underlying.Error(), "xdr: ")
	return fmt.Sprintf("xdr: %s (at %s)", uerr, err.Path)
}

func WithFieldError(err error, parts ...string) error {
	if err == nil {
		return nil
	}

	var combined string
	if parts[0] == "" {
		parts[0] = "<anonymous>"
	}

	switch len(parts) {
	case 1:
		combined = parts[0]
	case 3:
		combined = fmt.Sprintf("%s.%s(%s)", parts[0], parts[1], parts[2])
	default:
		combined = strings.Join(parts, ".")
	}

	switch err := err.(type) {
	case FieldError:
		err.Path = fmt.Sprintf("%s %s", combined, err.Path)
		return err
	default:
		return FieldError{err, combined}
	}
}
