// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

package tags

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// XDRTag represents a decoded XDR struct tag. It is a sequence of tag entries, where
// each one applies to the corresponding "layer" when multiple Go type definitions are
// nested on a field.
//
// As an example/clarification, consider the struct field:
//    Foo *[]byte `xdr:"opt/len:4"`
//
// This contains two layers of type:
//    * The pointer, with option "opt"
//    * The []byte slice, with option "len:4" indicating this is a 4 byte fixed size array
//
// There are three kinds of entry:
//   * Those which are flags, e.g. opt, opaque. These are encoded as a single byte
//   * Those which are setting parameters with values, such as len/maxlen. These are encoded
//     as a single flag byte followed by the parameter encoded as 32-bit integer
//   * Those which are setting multi-valued parameters, such as discriminant. These are encoded
//     as a single byte, followed by the number of parameters encoded as a 32-bit integer,
//     followed by that many parameters each encoded as a 32-bit integer
//
// The type of an entry is encoded in the most significant bits:
//   0b00 Tag, no value
//   0b01 Reserved
//   0b10 Tag with single 32-bit value (immediately following)
//   0b11 Tag with multiple values. A 32-bit value follows encoding the number of values, and
//        further values follow that
//
// As a somewhat special case, the various union tags relate to the definition of the enclosing
// structure
//
// We encode this into a byte slice so that it may be converted into a string and used as a part
// of a map key for codec resolution
type XDRTag []byte
type XDRTagKind byte

const (
	// Kinds without value, starting at 0x00 (0b00xx_xxxx)

	// No-op tag, sometimes required to handle layers of indirection
	// Encoded whenever two commas are encountered with no intervening value
	//
	// A tag must not contain trailing noops; by extension, a tag must not be composed
	// only of noops. They are only to be inserted when it is necessary to skip a level.
	Noop XDRTagKind = 0x00 | iota
	// Skip encoding this field (must be the only tag); Go struct tag `xdr:"-"`
	Skip
	// Indicates this field (which must be a pointer) is an optional field, i.e. was specified
	// with an * in the original XDR.
	Opt
	// Indicates this field (which must be a byte which is an array member) is to be encoded without
	// padding, i.e. was specified as `opaque` in the original XDR
	Opaque
	// Indicates that this field (which must be integral, and also the first member of the enclosing
	// type) is a union discriminant (switch), and that the enclosing struct represents an XDR union
	UnionSwitch
	// Indicates that this field (which must be a member of a union) is used when the union discriminant
	// has an otherwise unspecified value
	UnionDefault

	// Kinds with single value, starting at 0x80 (0b10xx_xxxx)

	// Specifies that this field (which must be a a slice or string) is to be encoded as a fixed
	// length array of the length that follows, i.e. without any length prefix
	Len XDRTagKind = 0x80 | iota

	// Specifies that this field (which must be a slice or string) is to be encoded as a variable
	// length array with length of up to the amount that follows
	MaxLen

	// Kinds with multiple values, starting 0xC0 (0b11xx_xxxx)

	// Specifies that this field (which must be a member of a union) is used when the union discriminant
	// has any of the specified values
	UnionCases = 0xC0 | iota
)

// Empty returns if this tag is empty
func (t XDRTag) Empty() bool {
	return len(t) == 0
}

// Kind returns the kind of the tag
func (t XDRTag) Kind() XDRTagKind {
	if len(t) > 0 {
		return XDRTagKind(t[0])
	} else {
		return Noop
	}
}

// valAt returns the 32-bit value at offset `offs`
func (t XDRTag) valAt(offs int) uint32 {
	// Compiler bounds check hint; see golang.org/issue/14808 and the
	// encoding/binary source code
	_ = t[offs+3]
	return uint32(t[offs])<<24 | uint32(t[offs+1])<<16 | uint32(t[offs+2])<<8 + uint32(t[offs+3])
}

// thisLen returns the length (in bytes) of this tag (as encoded)
func (t XDRTag) thisLen() int {
	switch {
	case len(t) == 0:
		return 0
	case t[0] < 0x80:
		return 1
	case t[0] < 0xC0:
		return 5
	default: // t[0] >= C0
		return 5 + int(t.valAt(1))*4
	}
}

// Next returns the next tag in the sequence
func (t XDRTag) Next() XDRTag {
	l := t.thisLen()
	if len(t) == l {
		return XDRTag(nil)
	} else {
		return XDRTag(t[l:])
	}
}

// Returns the only value for single valued options
func (t XDRTag) OnlyValue() uint32 {
	return t.valAt(1)
}

// Returns the range of values to iterate through to find all values for this tag
//
// for i, n := t.ValueRange(); i < n; i++ {
//     v := t.Value(i)
//
// The lower bound is not guaranteed to be 0 (in fact for multi-value tags it will
// always be 1; this is a very minor optimisation)
//
// As is typical for ranges, it's lower-inclusive upper-exclusive
func (t XDRTag) ValueRange() (int, int) {
	switch {
	case t[0] < 0x80:
		return 0, 0
	case t[0] < 0xC0:
		// Compiler bounds check hint
		_ = t[4]
		return 0, 1
	default: // t[0] >= 0xC0
		max := 1 + int(t.valAt(1))
		// Compiler bounds check hint
		_ = t[4*max]
		return 1, max
	}
}

// Returns the value at index n (which must be in between the range returned by ValueRange)
func (t XDRTag) Value(n int) uint32 {
	return t.valAt(1 + 4*n)
}

// Appends a tag with the specified values to the end of the current tag set
func (t XDRTag) Append(k XDRTagKind, values ...uint32) XDRTag {
	switch {
	case k < 0x80:
		if len(values) != 0 {
			panic(fmt.Sprintf("Attempt to append valueless tag %x with values %v", k, values))
		}

		tb := append([]byte(t), byte(k))
		return XDRTag(tb)

	case k < 0xC0:
		if len(values) != 1 {
			panic(fmt.Sprintf("Attempt to append single-value tag %x with %d values (%v)",
				k, len(values), values))
		}

		v := values[0]
		tb := append([]byte(t), byte(k), byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
		return XDRTag(tb)

	default: // k > 0xC0
		tb := []byte(t)
		nv := uint32(len(values))
		tb = append(tb, byte(k), byte(nv>>24), byte(nv>>16), byte(nv>>8), byte(nv))
		for _, v := range values {
			tb = append(tb, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
		}
		return XDRTag(tb)
	}
}

// Prepends the specified tag to the beginning of the current tag set
func (t XDRTag) Prepend(k XDRTagKind, values ...uint32) XDRTag {
	var nt XDRTag
	nt = nt.Append(k, values...)
	nt = XDRTag(append([]byte(nt), []byte(t)...))
	return nt
}

// Trimmed returns this tag with any trailing xt_noops removed
func (t XDRTag) Trimmed() XDRTag {
	// e tracks the length of all of the tags explored so far, including
	// the current one (the end of the current tag)
	// mark tracks the length up to the last tag we explored which wasn't
	// an Noop
	var e, mark int

	for ct := t; !ct.Empty(); ct = ct.Next() {
		e += ct.thisLen()
		if ct.Kind() != Noop {
			mark = e
		}
	}

	return XDRTag(t[0:mark])
}

// Returns this tag list as a byte slice. It must not be modified
func (t XDRTag) Bytes() []byte {
	return []byte(t)
}

// Returns this tag list as a byte string
func (t XDRTag) ByteString() string {
	return string([]byte(t))
}

// Vaguely pretty prints this tag list (for debugging purposes)
func (t XDRTag) String() string {
	if t.Empty() {
		return "Noop<empty>"
	}

	s := fmt.Sprintf("[%x]", t.Kind())

	i, n := t.ValueRange()
	if i != n {
		pfx := "("
		for ; i < n; i++ {
			s = fmt.Sprintf("%s%s%08x", s, pfx, t.Value(i))
			pfx = ", "
		}
		s += ")"
	}

	nt := t.Next()
	if !nt.Empty() {
		s = fmt.Sprintf("%s;%s", s, nt)
	}

	return s
}

var (
	emptyTag = XDRTag(nil)
	skipTag  = XDRTag([]byte{byte(Skip)})
)

func validForUnionSwitch(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32:
		return true

	default:
		return false
	}
}

func canBeOpt(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Interface, reflect.Ptr:
		return true
	default:
		return false
	}
}

// Specifies whether or not we're parsing this type in the direct context of a union
// If this is initially set to MaybeInUnion, then it will be bound to either of the two
// possible values as soon as we find the first indicative tag. If it's one of the two
// definitive values, then we'll never modify it
type IsInUnion int

const (
	// We're possibly in a union (see if we parse a union switch - then we'll know we are)
	MaybeInUnion IsInUnion = iota
	// We're defintely not in a union
	NotInUnion
	// We're definitely in a union
	InUnion
)

// Parse a struct tag to be applied to the specified type
func ParseStructTag(
	t reflect.Type,
	rtag reflect.StructTag,
	isUnion *IsInUnion,
) (XDRTag, error) {
	return ParseTag(t, rtag.Get("xdr"), isUnion)
}

func parseU32(s string) (uint32, error) {
	u64, err := strconv.ParseUint(s, 0, 32)
	return uint32(u64), err
}

func parseU32s(s string) ([]uint32, error) {
	vals := strings.Split(s, ",")
	u32s := make([]uint32, 0, len(vals))
	for _, v := range vals {
		u32, err := parseU32(v)
		if err != nil {
			return nil, err
		}
		u32s = append(u32s, u32)
	}
	return u32s, nil
}

// Parses the body of an XDR tag
func ParseTag(
	t reflect.Type,
	stags string,
	isUnion *IsInUnion,
) (
	xt XDRTag,
	err error,
) {
	stags = strings.TrimSpace(stags)

	switch stags {
	case "-":
		return skipTag, nil
	}

	parts := strings.Split(stags, "/")

	// Handle union related matters. Union related tags are special because they
	// (a) always come first in the composite tag, and (b) don't relate to a specific
	// type in the stack (so we should not pop a type)
	if strings.HasPrefix(parts[0], "union:") {
		p := parts[0]
		parts = parts[1:]

		switch {
		case p == "union:switch":
			if *isUnion != MaybeInUnion {
				return xt, errors.New("Found field annotated with `union:switch` tag which is not legal in a struct which is not a union or already has a switch")
			}

			if !validForUnionSwitch(t) {
				return xt, fmt.Errorf("Type %s not legal for union switch", t)
			}

			*isUnion = InUnion
			xt = xt.Append(UnionSwitch)

		case *isUnion != InUnion:
			return xt, fmt.Errorf("'%s' union tag not valid as we are not inside a union", p)

		case p == "union:false":
			xt = xt.Append(UnionCases, 0)
		case p == "union:true":
			xt = xt.Append(UnionCases, 1)
		case p == "union:default":
			xt = xt.Append(UnionDefault)
		default:
			vals, err := parseU32s(strings.TrimPrefix(p, "union:"))
			if err != nil {
				return xt, fmt.Errorf("Parsing `union:` values: %v", err)
			}

			xt = xt.Append(UnionCases, vals...)
		}
	} else if *isUnion == InUnion {
		return xt, errors.New("Every field inside a union struct must have a `union:` leading tag")
	} else {
		*isUnion = NotInUnion
	}

	// Next we should handle each of the tags which may correspond to one or more layers of
	// types
	for i, n := 0, len(parts); i < n; i++ {
		p := strings.TrimSpace(parts[i])
		switch {
		case p == "":
			xt = xt.Append(Noop)

		case p == "opt":
			if !canBeOpt(t) {
				return xt, fmt.Errorf("Type %s cannot be 'opt'", t)
			}
			xt = xt.Append(Opt)

		case p == "opaque":
			// Special case: It's really awkward to have to type 'xdr:";opaque"' all the time
			// on byte slices, so for this one case we will automatically handle opaque on
			// []byte or [N] byte
			switch t.Kind() {
			case reflect.Array, reflect.Slice:
				xt = xt.Append(Noop)
				t = t.Elem()
			}

			switch t.Kind() {
			case reflect.Int8, reflect.Uint8:
				xt = xt.Append(Opaque)

			default:
				return xt, fmt.Errorf("'opaque' label applied to %s, but only applicable to bytes", t)
			}

		case strings.HasPrefix(p, "len:"):
			len, err := parseU32(p[4:])
			if err != nil {
				return xt, fmt.Errorf("Error parsing XDR `len:` tag: %v", err)
			}

			switch t.Kind() {
			case reflect.Array:
				return xt, fmt.Errorf("Cannot apply `len:` tag to an array; just specify length directly")
			case reflect.String, reflect.Slice:
				xt = xt.Append(Len, len)
			default:
				return xt, fmt.Errorf("Cannot apply `len:` tag to %s; must be slice, string or map", t)
			}

		case strings.HasPrefix(p, "maxlen:"):
			len, err := parseU32(p[7:])
			if err != nil {
				return xt, fmt.Errorf("Error parsing XDR `maxlen:` tag: %v", err)
			}

			switch t.Kind() {
			case reflect.String, reflect.Slice:
				xt = xt.Append(MaxLen, len)
			default:
				return xt, fmt.Errorf("Cannot apply `maxlen:` tag to %s; must be slice, string or map", t)
			}

		default:
			return xt, fmt.Errorf("Unknown XDR tag '%s'", p)
		}

		// Descend one level through the types
		if i+1 != n {
			switch t.Kind() {
			case reflect.Array, reflect.Map, reflect.Ptr, reflect.Slice:
				t = t.Elem()

			default:
				return xt, fmt.Errorf("Trailing tags (%v) after reaching type %s", parts[i:], t)
			}
		}
	}

	return xt.Trimmed(), nil
}
