# terr

[![Go Reference](https://pkg.go.dev/badge/github.com/alnvdl/terr.svg)](https://pkg.go.dev/github.com/alnvdl/terr)
[![Test workflow](https://github.com/alnvdl/terr/actions/workflows/test.yaml/badge.svg)](https://github.com/alnvdl/terr/actions/workflows/test.yaml)

terr (short for **t**raced **err**or) is a minimalistic package for adding
error tracing to Go 1.20+.

Go's error representation primitives introduced in Go 1.13[^1] are quite
sufficient, but the lack of tracing capabilities makes it hard to confidently
debug errors across layers in complex applications.

To help with that, terr fully embraces the native Go error handling paradigms,
but it adds two features:
- file and line information for tracing errors;
- the ability to print error trees using the `fmt` package with the `%@` verb;

This package introduces the concept of **traced errors**: a wrapper for errors
that includes the location where they were created (`errors.New`), passed along
(`return err`), wrapped (`%w`) or masked (`%v`). Traced errors keep track of
children traced errors that relate to them. An error is a traced error if it
was returned by one of the functions of this package.

Traced errors work seamlessly with `errors.Is`, `errors.As` and `errors.Unwrap`
just as if terr were not being used.

## Using terr
Without terr                   | With terr
-------------------------------|------------------------------
`errors.New("error")`          | `terr.Newf("error")`
`fmt.Errorf("error: %w", err)` | `terr.Newf("error: %w", err)`
`[return] err`                 | `terr.Trace(err)`
`[return] &CustomError{}`      | `terr.TraceWithLocation(&CustomError{}, ...)`

`terr.Newf` can receive multiple errors. In fact, it is just a very slim
wrapper around `fmt.Errorf`. Any traced error passed to `terr.Newf` will be
included in the traced error tree, regardless of the `fmt` verb used for it.

`terr.Trace` and `terr.TraceWithLocation` on the other hand do nothing with
the error they receives (no wrapping and no masking), but they add one level
to the error tracing tree.

To obtain the full trace, terr functions must be used consistently. If
`fmt.Errorf` is used at one point, the error tracing information will be reset
at that point, but Go's wrapped error tree will be preserved even in that case.

Examples are available showing all these functions in use[^2].

In the glorious day error tracing is added to Go, and assuming it gets done in
a way that respects error handling as defined in Go 1.13+,
[removing `terr` from a codebase](#getting-rid-of-terr) should be a matter of
replacing the `terr` function calls with the equivalent documented expressions.

### Printing errors
A traced error tree can be printed with the special `%@` formatting verb. An
example is available[^3].

`%@` prints the error tree in a tab-indented, multi-line representation. If a
custom format is needed (e.g., JSON), it is possible to implement a function
that walks the error tree and generates that tree in the desired format. See
the [next subsection](#walking-the-traced-error-tree).

### Tracing custom errors
Custom errors can be turned into traced errors as well by using
`terr.TraceWithLocation` in constructor functions. An example is available[^4].

### Walking the traced error tree
Starting with Go 1.20, wrapped errors are kept as a n-ary tree. terr works by
building a tree containing tracing information in parallel, leaving the Go
error tree untouched, as if terr weren't being used. Each traced error is thus
a node of this parallel traced error tree.

`terr.TraceTree(err) TracedError` can be used to obtain the root of an n-ary
traced error tree, which can be navigated using the following methods:
```go
type TracedError interface {
	Error() string
	Location() (string, int)
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

An example is available[^5].

### Adopting terr
Run the following commands in a folder to recursively adopt terr in all its
files (make sure `goimports`[^6] is installed first):
```sh
$ go get github.com/alnvdl/terr
$ gofmt -w -r 'errors.New(a) -> terr.Newf(a)' .
$ gofmt -w -r 'fmt.Errorf -> terr.Newf' .
$ goimports -w .
```

Adopting `terr.Trace` and `terr.TraceWithLocation` is harder, as it is
impossible to write a simple gofmt rewrite rule that works for all cases.
Therefore, `terr.Trace` and `terr.TraceWithLocation` have to be applied as
needed in a code base. A rough guideline would be:
- `return err` becomes `return terr.Trace(err)`;
- `return NewCustomErr()` requires an adaptation in `NewCustomErr` to use
  `terr.TraceWithLocation`. An example is available[^4].
  `return terr.TraceWithLocation(NewCustomErr())`.

### Getting rid of terr
Run the following commands in a folder to recursively get rid of terr in all
its files (make sure `goimports`[^6] is installed first):
```sh
$ gofmt -w -r 'terr.Newf(a) -> errors.New(a)' .
$ gofmt -w -r 'terr.Newf -> fmt.Errorf' .
$ gofmt -w -r 'terr.Trace(a) -> a' .
$ gofmt -w -r 'terr.TraceWithLocation(a, b, c) -> a' .
$ goimports -w .
$ go mod tidy
```

[^1]: https://go.dev/blog/go1.13-errors
[^2]: https://pkg.go.dev/github.com/alnvdl/terr#pkg-examples
[^3]: https://pkg.go.dev/github.com/alnvdl/terr#example-package
[^4]: https://pkg.go.dev/github.com/alnvdl/terr#example-TraceWithLocation
[^5]: https://pkg.go.dev/github.com/alnvdl/terr#example-TraceTree
[^6]: https://pkg.go.dev/golang.org/x/tools/cmd/goimports
