# Go XDR Codec
[![PkgGoDev](https://pkg.go.dev/badge/go.e43.eu/xdr)](https://pkg.go.dev/go.e43.eu/xdr)

XDR (External Data Representation) is a binary encoding format developed at Sun
Microsystems in the 80s. It's used in a variety of systems and protocols, 
perhaps most notably the RPC system (Variably known as ONC RPC - where ONC 
stands for Open Network Computing - or Sun RPC) that underpins NFS, NIS, and 
some other related protocols.

XDR is specified in [RFC 4506](https://tools.ietf.org/html/rfc4506), obsoleting
previous [RFC 1832](https://tools.ietf.org/html/rfc1832) and 
[RFC 1014](https://tools.ietf.org/html/rfc1014.html).

That RFC also specifies an XDR schema language. That language is not implemented
here; see the companion [xdrgen](https://go.e43.eu/xdrgen) utility.

## Compatibility
This package's version may be below 1.0, but the intention is to avoid any 
compatibility breaks in the package's interface

## Performance
Performance is considered a feature; the implementation of this package includes
many optimisations with the hope of avoiding the need to employ code generation
in real systems.

A short excerpt of the benchmark results: 
```
BenchmarkUnionStructsEncode/XDREncoderDiscard-8  	 1765423	       617 ns/op
BenchmarkUnionStructsEncode/GobEncoderDiscard-8  	  540541	      1921 ns/op
BenchmarkUnionStructsEncode/JSONEncoderDiscard-8 	  564619	      2063 ns/op
```

The most heavily optimised versions require the use of the `unsafe` package. You
may opt out of this by using the `nounsafe` build tag

## License
Released under the [ISC](COPYING) license