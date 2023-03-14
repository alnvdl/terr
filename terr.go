// Package terr implements a set of functions for tracing errors in Go 1.20+.
package terr

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// tracedError implements the error and ErrorTracer interfaces, while being
// compatible with functions from the "errors" and "fmt" package in the
// standard library by implementing Is, As, Unwrap and Format.
type tracedError struct {
	error
	location
	children []ErrorTracer
}

type location struct {
	file string
	line int
}

func getCallerLocation() location {
	_, file, line, _ := runtime.Caller(2)
	return location{file, line}
}

func newTracedError(err error, children []any, loc location) *tracedError {
	terr := &tracedError{err, loc, nil}
	for _, child := range children {
		if child, ok := child.(*tracedError); ok {
			terr.children = append(terr.children, child)
		}
	}
	return terr
}

// Is returns whether the error is another error for use with errors.Is.
func (e *tracedError) Is(target error) bool {
	return errors.Is(e.error, target)
}

// As returns the error as another error for use with errors.As.
func (e *tracedError) As(target any) bool {
	return errors.As(e.error, target)
}

// Unwrap returns the wrapped error for use with errors.Unwrap.
func (e *tracedError) Unwrap() error {
	return errors.Unwrap(e.error)
}

// Error implements the error interface.
func (e *tracedError) Error() string {
	return e.error.Error()
}

// Location implements the ErrorTracer interface.
func (e *tracedError) Location() (string, int) {
	return e.file, e.line
}

// Children implements the ErrorTracer interface.
func (e *tracedError) Children() []ErrorTracer {
	return e.children
}

// Format implements fmt.Formatter.
func (e *tracedError) Format(f fmt.State, verb rune) {
	if verb == '@' {
		fmt.Fprint(f, strings.Join(treeRepr(e, 0), "\n"))
		return
	}
	fmt.Fprintf(f, fmt.FormatString(f, verb), e.error)
}

// treeRepr returns a tab-indented, multi-line representation of a traced error
// tree rooted in err.
func treeRepr(err error, depth int) []string {
	var locations []string
	te := err.(*tracedError)
	// No need to check the cast was successful: treeRepr is only invoked
	// internally via tracedError.Format. If that pre-condition is ever
	// violated, a panic is warranted.
	file, line := te.Location()
	locations = append(locations, fmt.Sprintf("%s%s @ %s",
		strings.Repeat("\t", depth),
		te.Error(),
		fmt.Sprintf("%s:%d", file, line)))
	children := te.Children()
	for _, child := range children {
		locations = append(locations, treeRepr(child, depth+1)...)
	}
	return locations
}

// Newf works exactly like fmt.Errorf, but returns a traced error. All traced
// errors passed as formatting arguments are included as children, regardless
// of the formatting verbs used for these errors.
// This function is equivalent to fmt.Errorf("...", ...). If used without verbs
// and additional arguments, it is equivalent to errors.New("...").
func Newf(format string, a ...any) error {
	return newTracedError(fmt.Errorf(format, a...), a, getCallerLocation())
}

// A TraceOption allows customization of errors returned by the Trace function.
type TraceOption func(e *tracedError)

// WithLocation returns a traced error with the given location. This option can
// be used in custom error constructor functions, so they can return a traced
// error pointing at their callers.
func WithLocation(file string, line int) TraceOption {
	return func(e *tracedError) {
		e.location = location{file, line}
	}
}

// WithChildren returns a traced error with the given traced errors appended as
// children Non-traced errors are ignored. This option can be used in custom
// error constructor functions to define the children traced errors for a
// traced error.
func WithChildren(children []error) TraceOption {
	return func(e *tracedError) {
		for _, child := range children {
			if terr, ok := child.(*tracedError); ok {
				e.children = append(e.children, terr)
			}
		}
	}
}

// Trace returns a new traced error for err. If err is already a traced error,
// a new traced error will be returned containing err as a child traced error.
// opts is an optional series of TraceOptions to be applied to the traced
// error. No wrapping or masking takes place in this function.
func Trace(err error, opts ...TraceOption) error {
	if err == nil {
		return nil
	}
	terr := newTracedError(err, []any{err}, getCallerLocation())
	for _, opt := range opts {
		opt(terr)
	}
	return terr
}

// ErrorTracer is an object capable of tracing an error's location and possibly
// other related errors, forming an error tracing tree.
// Please note that implementing ErrorTracer is not enough to make an error
// a traced error: only errors returned by functions in this package are
// considered traced errors.
type ErrorTracer interface {
	// error is the underlying error.
	error
	// Location identifies the file and line where error was created, traced,
	// wrapped or masked.
	Location() (string, int)
	// Children returns other traced errors that were traced, wrapped or
	// masked by this traced error.
	Children() []ErrorTracer
}

// TraceTree returns the root of the n-ary error tracing tree for err. Returns
// nil if err is not a traced error. This function can be used to represent the
// error tracing tree using custom formats.
func TraceTree(err error) ErrorTracer {
	te, _ := err.(*tracedError)
	if te == nil {
		return nil
	}
	return te
}
