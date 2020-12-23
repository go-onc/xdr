// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package coder

import (
	"fmt"
	"reflect"

	xdrinterfaces "go.e43.eu/xdr/interfaces"
	"go.e43.eu/xdr/internal/errors"
	"go.e43.eu/xdr/internal/tags"
)

type structCodec struct {
	name   string
	fields []field
}

var _ xCodec = &structCodec{}

type switchKind byte

const (
	switchKindBool switchKind = iota
	switchKindInt
	switchKindUint
)

type unionCodec struct {
	name        string
	switchField field
	bodyFields  []field
	cases       map[uint32]int
	defaultCase int
	switchKind  switchKind
}

var _ xCodec = &unionCodec{}

func makeStructCodec(cr *Coder, t reflect.Type) xdrinterfaces.Codec {
	var (
		f   reflect.StructField
		tag tags.XDRTag
		err error
	)

	// Iterate until we figure out if we're a union or not
	isUnion := tags.MaybeInUnion
	i, fieldCount := 0, t.NumField()
	for ; i < fieldCount && isUnion == tags.MaybeInUnion; i++ {
		f = t.Field(i)
		tag, err = tags.ParseStructTag(f.Type, f.Tag, &isUnion)
		if err != nil {
			return &errorCodec{fmt.Errorf("Parsing tag of field '%s' of '%s': %v",
				f.Name, t, err)}
		}

		switch {
		case tag.Kind() == tags.Skip:
			continue
		case isUnion == tags.MaybeInUnion:
			// Should be unreachable
			panic("We found an unskipped field but somehow don't know if we're a union or not")
		}
	}

	switch isUnion {
	case tags.MaybeInUnion:
		// We never figured it out but also we didn't find any (unskipped) fields. This
		// is a degenerate empty case, so we'll just construct an empty struct codec
		return &structCodec{name: t.Name()}

	case tags.NotInUnion:
		// We're actually a struct
		c := &structCodec{
			name:   t.Name(),
			fields: make([]field, 0, fieldCount),
		}

		c.fields = append(c.fields, makeField(cr, f, tag))
		for ; i < fieldCount; i++ {
			f = t.Field(i)
			tag, err = tags.ParseStructTag(f.Type, f.Tag, &isUnion)
			if err != nil {
				return &errorCodec{fmt.Errorf("Parsing tag of field '%s' of '%s': %v",
					f.Name, t, err)}
			}

			if tag.Kind() == tags.Skip {
				continue
			}

			c.fields = append(c.fields, makeField(cr, f, tag))
		}

		return c

	case tags.InUnion:
		// We're acually a union, and f is our switch
		// Every following field is going to be prefixed by the xt_unioncases or xt_uniondefault tag
		if tag.Kind() != tags.UnionSwitch {
			// Shouldn't happen
			panic("First element of union not switch")
		}

		var switchKind switchKind
		switch f.Type.Kind() {
		case reflect.Int32:
			switchKind = switchKindInt

		case reflect.Uint32:
			switchKind = switchKindUint

		case reflect.Bool:
			switchKind = switchKindBool

		default:
			// Shouldn't happen - tag parsing should have validated legality
			panic("Switch field of union not valid (must be int32, uint32 or bool)")
		}

		c := &unionCodec{
			name:        t.Name(),
			switchField: makeField(cr, f, tag.Next()),
			bodyFields:  make([]field, fieldCount),
			cases:       make(map[uint32]int, fieldCount-1),
			defaultCase: -1,
			switchKind:  switchKind,
		}

		for ; i < fieldCount; i++ {
			f = t.Field(i)
			tag, err = tags.ParseStructTag(f.Type, f.Tag, &isUnion)
			if err != nil {
				return &errorCodec{fmt.Errorf("Parsing tag of field '%s' of '%s': %v",
					f.Name, t, err)}
			}

			if tag.Kind() == tags.Skip {
				continue
			}

			c.bodyFields[i] = makeField(cr, f, tag.Next())

			switch tag.Kind() {
			case tags.UnionCases:
				for j, e := tag.ValueRange(); j < e; j++ {
					v := tag.Value(j)
					if _, ok := c.cases[v]; ok {
						return &errorCodec{fmt.Errorf("Union value 0x%08x of %s duplicated", v, t)}
					}
					c.cases[v] = i
				}

			case tags.UnionDefault:
				if c.defaultCase != -1 {
					return &errorCodec{fmt.Errorf("Default case of %s duplicated", t)}
				}
				c.defaultCase = i
			}
		}

		return c

	default:
		panic("unreachable")
	}
}

func (c *structCodec) encodeReflect(e xdrinterfaces.Encoder, v reflect.Value) error {
	for _, f := range c.fields {
		_, err := f.encode(e, v)
		if err != nil {
			return errors.WithFieldError(err, c.name, f.name)
		}
	}
	return nil
}

func (c *unionCodec) encodeReflect(e xdrinterfaces.Encoder, v reflect.Value) (err error) {
	swv, err := c.switchField.encode(e, v)
	if err != nil {
		err = errors.WithFieldError(err, c.name, c.switchField.name, "union:switch")
		return
	}

	var swVal uint32
	switch c.switchKind {
	case switchKindBool:
		if swv.Bool() {
			swVal = 1
		}
	case switchKindUint:
		swVal = uint32(swv.Uint())
	default: //switchKindInt
		swVal = uint32(swv.Int())
	}

	caseField, exists := c.cases[swVal]
	if !exists {
		caseField = c.defaultCase
	}

	if caseField == -1 {
		err = errors.ErrUnionSwitchArmUndefined
		return errors.WithFieldError(err, c.name, "?", fmt.Sprintf("union:0x%x", caseField))
	}

	f := c.bodyFields[caseField]
	_, err = f.encode(e, v)
	if err != nil {
		err = errors.WithFieldError(err, c.name, f.name, fmt.Sprintf("union:0x%x", swVal))
	}
	return
}
