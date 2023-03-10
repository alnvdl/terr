// Package terr implements a minimalistic library for adding error tracing in
// Go 1.20+.
package terr

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// TracedError is a wrapper for error that can be used to keep a tree of
// tracing information for related errors.
type TracedError interface {
	// error is the underlying error.
	error
	// Location returns a string in the format "file:line" pointing to the
	// location in the code where the error was created, traced, wrapped or
	// masked.
	Location() string
	// Children returns other traced errors that were traced, wrapped or
	// masked by this traced error.
	Children() []TracedError
}

// tracedError is an error with a file:line location and pointer to the
// traced errors that precedes it in the chain (if any).
type tracedError struct {
	err   error
	loc   string
	terrs []TracedError
}

// newTracedError builds a traced error for err and its children traced errors
// (whether passed, wrapped or masked).
func newTracedError(err error, children []error) error {
	var terrs []TracedError
	for _, child := range children {
		if terr, ok := child.(*tracedError); ok {
			terrs = append(terrs, terr)
		}
	}

	return &tracedError{
		err:   err,
		loc:   getLocation(2),
		terrs: terrs,
	}
}

func getLocation(depth int) string {
	_, file, line, _ := runtime.Caller(depth + 1)
	return fmt.Sprintf("%s:%d", file, line)
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
	return errors.Is(e.err, target)
}

// As returns the error as another error for use with errors.As.
func (e *tracedError) As(target interface{}) bool {
	return errors.As(e.err, target)
}

// Unwrap returns the wrapped error for use with errors.Unwrap.
func (e *tracedError) Unwrap() error {
	return errors.Unwrap(e.err)
}

// Error implements the error interface.
func (e *tracedError) Error() string {
	return e.err.Error()
}

// Location implements the TracedError interface.
func (e *tracedError) Location() string {
	return e.loc
}

// Children implements the TracedError interface.
func (e *tracedError) Children() []TracedError {
	return e.terrs
}

// Format implements fmt.Formatter.
func (e *tracedError) Format(f fmt.State, verb rune) {
	if verb == '@' {
		fmt.Fprint(f, strings.Join(treeRepr(e, 0), "\n"))
		return
	}
	fmt.Fprintf(f, fmt.FormatString(f, verb), e.err)
}

// treeRepr returns human-readable lines representing an error tree rooted in
// err.
func treeRepr(err error, depth int) []string {
	var locations []string
	te := err.(TracedError)
	// No need to check the cast was successful: treeRepr is only invoked
	// internally via tracedError.Format. If that pre-condition is ever
	// violated, a panic is warranted.
	locations = append(locations, fmt.Sprintf("%s%s @ %s",
		strings.Repeat("\t", depth),
		te.Error(),
		te.Location()))
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
	return newTracedError(fmt.Errorf(format, a...), filterErrors(a))
}

// Trace returns a new traced error for err. If err is already a traced error,
// a new traced error will be returned containing err as a child traced error.
// No wrapping or masking takes place in this function.
func Trace(err error) error {
	if err == nil {
		return nil
	}
	return newTracedError(err, []error{err})
}

// TraceTree returns the root of the n-ary traced error tree for err. Returns
// nil if err is nil.
// Presenting these arbitrarily complex error trees in human-comprehensible way
// is left as an exercise to the caller. Or just use fmt.Sprintf("%@", err) for
// a tab-indented multi-line string representation of the tree.
func TraceTree(err error) TracedError {
	te, _ := err.(TracedError)
	return te
}
