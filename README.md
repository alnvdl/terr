# terr

[![Go Reference](https://pkg.go.dev/badge/github.com/alnvdl/terr.svg)](https://pkg.go.dev/github.com/alnvdl/terr)
[![Test workflow](https://github.com/alnvdl/terr/actions/workflows/test.yaml/badge.svg)](https://github.com/alnvdl/terr/actions/workflows/test.yaml)

terr (short for **t**raced **err**or) is a minimalistic library for adding
error tracing to Go 1.20+.

The error representation primitives introduced in Go 1.13[^1] are quite
sufficient, but the lack of tracing capabilities makes it hard to confidently
debug errors across layers in complex applications.

To help with that, terr fully embraces the native Go error handling paradigms,
but it adds two features:
- file and line information for tracing errors;
- the ability to print error tracing trees using the `fmt` package with the
  `%@` verb;

This package introduces the concept of **traced errors**: a wrapper for errors
that includes the location where they were created (`errors.New`), passed along
(`return err`), wrapped (`%w`) or masked (`%v`). Traced errors keep track of
children traced errors that relate to them. An error is a traced error if it
was returned by one of the functions of this package.

Most importantnly, traced errors work seamlessly with `errors.Is`, `errors.As`
and `errors.Unwrap` just as if terr were not being used.

## Using terr
Without terr                   | With terr
-------------------------------|------------------------------
`errors.New("error")`          | `terr.Newf("error")`
`fmt.Errorf("error: %w", err)` | `terr.Newf("error: %w", err)`
`[return] err`                 | `terr.Trace(err)`
`[return] &CustomError{}`      | `terr.TraceSkip(&CustomError{}, 1)`

`terr.Newf` can receive multiple errors. In fact, it is just a very slim
wrapper around `fmt.Errorf`. Any traced error passed to `terr.Newf` will be
included in the error tracing tree, regardless of the `fmt` verb used for it.

`terr.Trace` and `terr.TraceSkip` on the other hand do nothing with the error
they receive (no wrapping and no masking), but they add one level to the error
tracing tree. `terr.TraceSkip` lets custom errors constructors return a traced
error with the location defined by skipping a number of stack frames.

To obtain the full trace, terr functions must be used consistently. If
`fmt.Errorf` is used at one point, the error tracing information will be reset
at that point, but Go's wrapped error tree will be preserved even in that case.

Examples are available showing these functions in use[^2].

In the glorious day error tracing is added to Go, and assuming it gets done in
a way that respects error handling as defined in Go 1.13+,
[removing `terr` from a codebase](#getting-rid-of-terr) should be a matter of
replacing the `terr` function calls with equivalent expressions.

### Tracing custom errors
Constructor functions for custom error types and wrapped sentinel errors[^1]
can use `terr.TraceSkip(err, skip)`. An example is available[^3].

### Printing errors
An error tracing tree can be printed with the special `%@` formatting verb. An
example is available[^4].

`%@` prints the tree in a tab-indented, multi-line representation. If a custom
format is needed (e.g., JSON), it is possible to implement a function that
walks the error tracing tree and outputs it in the desired format. See the
[next subsection](#walking-the-error-tracing-tree).

### Walking the error tracing tree
Starting with Go 1.20, wrapped errors are kept as a n-ary tree. terr works by
building a tree containing tracing information in parallel, leaving the Go
error tree untouched, as if terr were not being used. Each traced error is thus
a node in this parallel error tracing tree.

`terr.TraceTree(err) ErrorTracer` can be used to obtain the root of an n-ary
error tracing tree, which can be navigated using the following methods:
```go
type ErrorTracer interface {
	Error() string
	Location() (string, int)
	Children() []ErrorTracer
}
```

Note that this is **not** the tree of wrapped errors built by the Go standard
library, because:
- if non-traced errors are provided to `terr.Newf`, even if wrapped, they will
  not be a part of the error tracing tree;
- even masked (`%v`) errors will be part of the error tracing tree if
  `terr.Newf` was used to mask them.

Methods provided by the by the Go standard library should be used to walk Go's
wrapped error tree, which would include non-traced errors and ignore masked
errors (e.g., `errors.Unwrap`).

An example is available[^5].

### Adopting terr
Adopting terr requires some thought about how errors are being constructed and
which errors are worth tracing. Usage of terr may vary greatly for different
code bases, and it is easiest to adopt terr when just the vanilla Go error
handling practices are in use, without any third-party libraries.

The terr examples[^2], particularly the example for `terr.TraceSkip`[^3],
provide a good illustration of how to use terr while following Go's
recommended error guidelines[^1].

In larger code bases, using `gofmt -r` might help, but it might also produce
unwanted results if not used carefully. Applying the reverse of the rewrite
rules mentioned in the [next sub-section](#getting-rid-of-terr) may be helpful
in some cases.

### Getting rid of terr
While adopting terr can be hard, removing it from a code base is very easy.
Run the following commands in a directory tree to get rid of terr in all its
files (make sure `goimports`[^6] is installed first):
```sh
$ gofmt -w -r 'terr.Newf(a) -> errors.New(a)' .
$ gofmt -w -r 'terr.Newf -> fmt.Errorf' .
$ gofmt -w -r 'terr.Trace(a) -> a' .
$ gofmt -w -r 'terr.TraceSkip(a, b) -> a' .
$ goimports -w .
$ go mod tidy
```

Also make sure to remove any uses of the `%@` fmt verb, since this verb only
works with traced errors.

[^1]: https://go.dev/blog/go1.13-errors
[^2]: https://pkg.go.dev/github.com/alnvdl/terr#pkg-examples
[^3]: https://pkg.go.dev/github.com/alnvdl/terr#example-TraceSkip
[^4]: https://pkg.go.dev/github.com/alnvdl/terr#example-package
[^5]: https://pkg.go.dev/github.com/alnvdl/terr#example-TraceTree
[^6]: https://pkg.go.dev/golang.org/x/tools/cmd/goimports
