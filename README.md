# terr

[![Go Reference](https://pkg.go.dev/badge/github.com/alnvdl/terr.svg)](https://pkg.go.dev/github.com/alnvdl/terr)
[![Test workflow](https://github.com/alnvdl/terr/actions/workflows/test.yaml/badge.svg)](https://github.com/alnvdl/terr/actions/workflows/test.yaml)


terr (short for **t**raced **err**or) is a minimalistic library for adding
error tracing in Go 1.20+.

It was developed because Go's error handling primitives introduced in Go 1.13
are quite sufficient, but the lack of tracing capabilities makes it really hard
to confidently chase problems across layers in complex applications.

So terr fully embraces the
[changes to errors introduced in Go 1.13](https://go.dev/blog/go1.13-errors),
but it adds two extra features:
- file and line information for tracing errors;
- the ability to print error trees using the `fmt` package and the `%@` verb;

Starting with Go 1.20, wrapped errors are kept as a n-ary tree. terr works by
build a parallel tree containing tracing information, leaving the Go error tree
untouched, as if terr weren't being used.

This library introduces the concept of **traced errors**: a wrapper for regular
errors that includes the location of where they were created (`errors.New`),
passed along (`return err`), wrapped (`%w`) or masked (`%v`). Traced errors
keep track of children traced errors that relate to them. Each traced error
is thus a node of the parallel traced error tree. An error is considered a
traced error if it was returned by one of the functions of this library.

Traced errors work seamlessly with `errors.Is`, `errors.As` and `errors.Unwrap`
just as if terr were not being used. All of the patterns outlined in the
[Go 1.13 error handling blog post](https://go.dev/blog/go1.13-errors) are
supported, but with the addition of error tracing.

In the glorious day Golang introduces error tracing, and assuming it gets done
in a way that respects error handling as defined in Go 1.13+,
[removing `terr` from a codebase](#getting-rid-of-terr) should be a matter of
replacing the `terr` function calls with equivalent documented expressions:

Without terr                   | With terr
-------------------------------|------------------------------
`errors.New("error")`          | `terr.Newf("error")`
`fmt.Errorf("error: %w", err)` | `terr.Newf("error: %w", err)`
`[return] err`                 | `terr.Trace(err)` (annotates file and line)

To obtain the full trace, terr must be used consistently. If `fmt.Errorf` is
used at one point, the error tracing information will be reset at that point
(but Go's wrapped error tree will be preserved even in that case).

`terr.Newf` can wrap multiple errors. In fact, it is just a very slim wrapper
around `fmt.Errorf`. Any traced error passed to `terr.Newf` will be included in
the traced error tree, regardless of the `fmt` verb used.

## Example
The following code:
```go
err := terr.Newf("base")
traced := terr.Trace(err)
wrapped := terr.Newf("wrapped: %w", traced)
masked := terr.Newf("masked: %v", wrapped)
fmt.Printf("%@\n", masked)
```

Will output:
```
masked: wrapped: base @ /mygomod/file.go:14
        wrapped: base @ /mygomod/file.go:13
                base @ /mygomod/file.go:12
                        base @ /mygomod/file.go:11
```

## Printing errors
A traced error tree can be printed with the special `%@` formatting verb.
```go
fmt.Printf("%@\n", terr.Newf("problem: %w", terr.Newf("error")))
```

The code above will print the error tree in an tab-indented multi-line
representation. If a custom format is needed (e.g., JSON), it is possible to
implement a function that walks the error tree and generates that tree in the
desired format. See the [next subsection](#walking-the-traced-error-tree).

## Walking the traced error tree
`terr.TraceTree(err) TracedError` can be used to obtain the root of an n-ary
traced error tree, which can be navigated using the following methods:
```go
type TracedError interface {
	Error() string
	Location() string
	Children() []TracedError
}
```

Note that this is **not** the tree of wrapped errors built by Go's standard
library, because:
- if non-traced errors are provided to `terr.Newf`, even if wrapped, they will
  not be a part of the traced error tree;
- even masked (`%v`) errors will be part of the traced error tree if
  `terr.Newf` was used to mask them.


Methods provided by the by the Go standard library should be used to walk Go's
wrapped error tree, which would includes non-traced errors and ignores masked
errors (e.g., `errors.Unwrap`).

## Adopting terr
Run the following commands in a folder to recursively adopt terr in all its
files (make sure [`goimports`](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)
is installed first):
```sh
$ go get github.com/alnvdl/terr
$ gofmt -w -r 'errors.New(a) -> terr.Newf(a)' .
$ gofmt -w -r 'fmt.Errorf -> terr.Newf' .
$ goimports -w .
```

Adopting `terr.Trace` is harder, as it's impossible write a simple gofmt
rewrite rule that works for all cases. `terr.Trace` has to be applied as needed
in a code base, typically in cases where `return err` is used, turning it
into `return terr.Trace(err)`.

## Getting rid of terr
Run the following commands in a folder to recursively get rid of terr in all
its files (make sure [`goimports`](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)
is installed first):
```sh
$ gofmt -w -r 'terr.Newf(a) -> errors.New(a)' .
$ gofmt -w -r 'terr.Newf -> fmt.Errorf' .
$ gofmt -w -r 'terr.Trace(a) -> a' .
$ goimports -w .
$ go mod tidy
```

