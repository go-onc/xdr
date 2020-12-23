// Copyright 2020 Erin Shepherd
// SPDX-License-Identifier: ISC

// Package xdr implements encoding and decoding of the XDR
// (External Data Representation) format, as specified in RFC 4506.
//
// The Encoder/Decoder types in this package offer low level marshalling
// functions, but in most cases you will wish to use the higher level fuctions
// based upon reflection.
//
// The mapping from Go types to XDR is:
//
//                        Go | XDR
//     ----------------------+--------------------
//                      bool | bool
//      int8,  int16,  int32 | int
//     uint8, uint16, uint32 | unsigned int
//                     int64 | hyper
//                    uint64 | unsigned hyper
//                   float32 | float
//                   float64 | double
//                 complex64 | struct { float  Re; float  Im; }
//                complex128 | struct { double Re; double Im; }
//                    string | string ident<>
//                        *T | T (Go pointers are ignored)
//                       []T | T ident<>
//                      [N]T | T ident[N]
//                  struct{} | void
//              struct{ ...} | struct { ... }
//
// (in the above and following, `T`, `N` and `ident` are metavariables corresponding to
// an arbitrary type, an arbitrary (maximum) length, and an arbitrary identifier
// respectively)
//
// Go has no direct equivalent of XDR enumerations; therefore they should be defined
// as type aliases for uint32:
//
//     type MyEnum uint32
//
// There are some XDR types which cannot be expressed with just these; therefore
// additional control is provided using the `xdr:"..."` struct tag:
//
//                 XDR | Go
//     ----------------+--------------------------------
//     T *ident        |  *T     `xdr:"opt"`
//     T ident<N>      | []T     `xdr:"maxlen:N"`
//     string ident[N] | string  `xdr:"len:N"`
//     string ident<N> | string  `xdr:"maxlen:N"`
//     opaque ident<>  | []byte  `xdr:"opaque"`
//     opaque ident[N] | [N]byte `xdr:"opaque"`
//     opaque ident<N> | []byte  `xdr:"maxlen:N/opaque"`
//
// Some structure field definitions contain multiple layers of types. For example, the type
// *T can be considered as having two layers (ptr t), while the type *[]T has three (ptr slice T).
// We may need or wish to apply tags to modify how a type is marshalled at each layer, and need
// to avoid confusion as to which layer a tag applies.
//
// Tags are therefore applied heirarchically: tags separated by forward slashes and
// specified left to right apply in turn from the outer to the inner type. Tags are always applied
// to the type at the corresponding level; if it is necessary to skip a level, then that level should
// be left empty.
//
// Defined tags:
//
//     `-`
//         Must comprise the entirety of the tag; indicates that the field is to be skipped
//         from XDR (un)marshalling
//
//     `opt`
//         Applied to a pointer or interface type: When encoding, a bool indicating the
//         presence or absence of a value is written followed by the value (if present)
//
//         XDR: T *ident
//         Go:  ident *T `xdr:opt`
//
//     `opaque`
//         Applied to `byte` type as a member of a slice or array, indicates that the body
//         of the slice should be encoded densely (1 byte per element instead of four). Applied
//         to the byte slice or byte array itself, it has the same meaning; the following pairs are
//         equivalent:
//           F  []byte  `xdr:"opaque"` | F  []byte `xdr:"/opaque"`
//           F [N]byte  `xdr:"opaque"` | F [N]byte `xdr:"/opaque"`
//         (Both methods are permitted to allow other modifiers - such as `len:N` or `maxlen:N` - to
//         be applied to the enclosing type without conflicting, while avoiding the need to include
//         the leading slash in simple cases)
//
//         XDR: opaque ident[N]               opaque ident<N>
//         Go:  ident [N]byte `xdr:"opaque"`  ident []byte `xdr:"maxlen:N/opaque"`
//
//     `len:N`
//         Only applicable to strings, specifies that this string  is to be encoded as fixed width
//
//         Example: ident string `xdr:"len:16"`
//
//     `maxlen:N`
//         Only applicable to strings or slices, specifies a maximum permitted length
//
//         Example: ident string `xdr:"maxlen:16"`
//
// Unions are slightly more tricky to define: Go does not provide a direct analogue for XDR unions.
// Instead, define a struct where the fields are annotated with union tags:
//
//                                           XDR | Go
//     ------------------------------------------+-------------------------------------------
//     union my_union switch(int switch_field) { | type MyUnion struct {
//                                               |   SwitchField  int32  `xdr:"union:switch"`
//       case 0:  type_a  field_a;               |   FieldA       *TypeA `xdr:"union:0"`
//       case 1:  type_b *field_b;               |   FieldB       *TypeB `xdr:"union:1/opt"`
//       default: type_c  field_default;         |   FieldDefault *TypeC `xdr:"union:default"`
//     }                                         | }
//
//     `union:switch`
//          Specifies that the enclosing structure is a union, and that this field is the
//          switch. The field must be of type int32, uint32 or bool.
//
//          Must be specified on the first field within the struct which is not skipped using
//          `-`. If specified, every field must have a case tag
//
//     `union:A,B,C`, `union:true`, `union:false`, `union:default`
//          Specifies which case(s) this field corresponds to. A/B/C are must be numeric values
//          (unfortunately constants are not supported). `true` and `false` may be used instead
//          for boolean switch fields. `default` specifies this is the default case (if no other
//          case was encountered)
//
// Union tags bind to the enclosing structure type; in this regard, they are a special case. They
// may be followed by type-related specifiers like normal.
//
// You can specify custom behaviour for your type using the Marshaler interface. If implemented,
// it replaces the default behaviour. You can override behaviour for third party types by
// implementing and regisering a Codec; see the documentation for that type and the Coder with
// which they are registered.
//
// To avoid confusion and conflicts between different packages, it is not possible to register new
// codecs with the default (global) Coder.
package xdr

import xdrinterfaces "go.e43.eu/xdr/interfaces"

// interface Coder is the top-level interface to the XDR library
//
// A coder (which may be safely used from multiple threads) provides the ability
// to marshal objects to and from XDR. It also contains a repository of Codecs
// which know how to marshal various types
type Coder = xdrinterfaces.Coder

// interface Encoder is the interface to the XDR encoder
type Encoder = xdrinterfaces.Encoder

// interface Decoder is the interface to the XDR decoder
type Decoder = xdrinterfaces.Decoder
