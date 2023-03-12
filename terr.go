// Package terr implements a minimalistic library for adding error tracing in
// Go 1.20+.
package terr

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// TracedError is an error with tracing information (its location) and possibly
// other related errors, forming a tree of traced errors.
type TracedError interface {
	// error is the underlying error.
	error
	// Location identifies the file and line where error was created, traced,
	// wrapped or masked.
	Location() (string, int)
	// Children returns other traced errors that were traced, wrapped or
	// masked by this traced error.
	Children() []TracedError
}

// tracedError implements the TracedError interface while being compatible with
// functions from the "errors" package in the standard library.
type tracedError struct {
	error
	location
	children []TracedError
}

type location struct {
	file string
	line int
}

func getCallerLocation() location {
	_, file, line, _ := runtime.Caller(2)
	return location{file, line}
}

func newTracedError(err error, children []error, loc location) error {
	var terrs []TracedError
	for _, child := range children {
		if terr, ok := child.(TracedError); ok {
			terrs = append(terrs, terr)
		}
	}
	return &tracedError{err, loc, terrs}
}

func filterErrors(objs []interface{}) []error {
	var errors []error
	for _, o := range objs {
		if err, ok := o.(error); ok {
			errors = append(errors, err)
		}
	}
	return errors
}

// Is returns whether the error is another error for use with errors.Is.
func (e *tracedError) Is(target error) bool {
	return errors.Is(e.error, target)
}

// As returns the error as another error for use with errors.As.
func (e *tracedError) As(target interface{}) bool {
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

// Location implements the TracedError interface.
func (e *tracedError) Location() (string, int) {
	return e.file, e.line
}

// Children implements the TracedError interface.
func (e *tracedError) Children() []TracedError {
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

// treeRepr returns a tab-indented, multi-line representation of an error tree
// rooted in err.
func treeRepr(err error, depth int) []string {
	var locations []string
	te := err.(TracedError)
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
// Implements the pattern fmt.Errorf("...", ...). If used without verbs and
// additional arguments, it also implements the pattern errors.New("...").
func Newf(format string, a ...interface{}) error {
	return newTracedError(
		fmt.Errorf(format, a...),
		filterErrors(a),
		getCallerLocation(),
	)
}

// Trace returns a new traced error for err. If err is already a traced error,
// a new traced error will be returned containing err as a child traced error.
// No wrapping or masking takes place in this function.
func Trace(err error) error {
	if err == nil {
		return nil
	}
	return newTracedError(err, []error{err}, getCallerLocation())
}

// TraceWithLocation works like Trace, but lets the caller specify a file and
// line for the error. This is most useful for custom error constructor
// functions, so they can return a traced error pointing at their callers.
func TraceWithLocation(err error, file string, line int) error {
	if err == nil {
		return nil
	}
	return newTracedError(err, []error{err}, location{file, line})
}

// TraceTree returns the root of the n-ary traced error tree for err. Returns
// nil if err is nil or not a traced error.
// Presenting these arbitrarily complex error trees in human-comprehensible way
// is left as an exercise to the caller. Or just use fmt.Sprintf("%@", err) for
// a tab-indented multi-line string representation of the tree.
func TraceTree(err error) TracedError {
	te, _ := err.(TracedError)
	return te
}
