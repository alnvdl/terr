package terr_test

import (
	"errors"
	"fmt"

	"github.com/alnvdl/terr"
)

// This example shows how to combine different terr functions and print an
// error tracing tree at the end.
func Example() {
	err := terr.Newf("base")
	traced := terr.Trace(err)
	wrapped := terr.Newf("wrapped: %w", traced)
	masked := terr.Newf("masked: %v", wrapped)
	fmt.Printf("%@\n", masked)
}

// This example shows how Newf interacts with traced and non-traced errors.
// Traced errors are included in the trace regardless of the fmt verb used for
// them, while non-traced errors are handled as fmt.Errorf would, but they do
// not get included in the trace.
func ExampleNewf() {
	nonTracedErr := errors.New("non-traced")
	tracedErr1 := terr.Newf("traced 1")
	tracedErr2 := terr.Newf("traced 2")
	newErr := terr.Newf("errors: %w, %v, %w",
		nonTracedErr,
		tracedErr1,
		tracedErr2)

	fmt.Printf("%@\n", newErr)
	fmt.Println("---")

	// errors.Is works.
	fmt.Println("newErr is nonTracedErr:", errors.Is(newErr, nonTracedErr))
	fmt.Println("newErr is tracedErr1:", errors.Is(newErr, tracedErr1))
	fmt.Println("newErr is tracedErr2:", errors.Is(newErr, tracedErr2))
}

// This example shows how Trace interacts with traced and non-traced errors.
func ExampleTrace() {
	// Adds tracing information to non-traced errors.
	nonTracedErr := errors.New("non-traced")
	fmt.Printf("%@\n", terr.Trace(nonTracedErr))
	fmt.Println("---")
	// Adds another level of tracing information to traced errors.
	tracedErr := terr.Newf("traced")
	fmt.Printf("%@\n", terr.Trace(tracedErr))
}

var ErrConnection = errors.New("connection error")

func connectionError(text string) error {
	return terr.TraceSkip(fmt.Errorf("%w: %s", ErrConnection, text), 1)
}

var ErrValidation = errors.New("validation error")

type ValidationError struct{ field, msg string }

func (e *ValidationError) Error() string {
	return e.msg
}

func NewValidationError(field, msg string) error {
	return terr.TraceSkip(&ValidationError{field, msg}, 1)
}

// This example shows how to use TraceSkip to add tracing when using two common
// patterns in error constructors: wrapped sentinel errors and custom error
// types. TraceSkip works by accepting a number of stack frames to skip when
// defining the location of the traced errors it returns. In this example, the
// location is being set to the location of the callers of the error
// constructors, and not the constructors themselves.
func ExampleTraceSkip() {
	// It is considered a good practice in Go to never return sentinel errors
	// directly, but rather to wrap them like we do with connectionError here,
	// so they can be turned into custom errors later if needed, without
	// breaking callers in the process.
	// err1 will be annotated with the line number of the following line.
	err1 := connectionError("timeout")
	fmt.Printf("%@\n", err1)

	// errors.Is works.
	fmt.Println("\tIs ErrConnection:", errors.Is(err1, ErrConnection))
	fmt.Println("---")

	// If an error needs to include more details, a custom error type needs to
	// be used.
	// err2 will be annotated with the line number of the following line.
	err2 := NewValidationError("x", "x must be >= 0")
	fmt.Printf("%@\n", err2)

	// errors.As works.
	var customErr *ValidationError
	ok := errors.As(err2, &customErr)
	fmt.Println("\tIs ValidationError:", ok)
	fmt.Println("\tCustom error field:", customErr.field)
	fmt.Println("\tCustom error message:", customErr.msg)
}

// This example shows how to use the n-ary error tracing tree returned by
// TraceTree.
func ExampleTraceTree() {
	nonTracedErr := errors.New("non-traced")
	tracedErr1 := terr.Newf("traced 1")
	tracedErr2 := terr.Newf("traced 2")
	newErr := terr.Newf("%w, %v, %w",
		nonTracedErr,
		tracedErr1,
		tracedErr2)

	printNode := func(node terr.ErrorTracer) {
		fmt.Printf("Error: %v\n", node.Error())
		file, line := node.Location()
		fmt.Printf("Location: %s:%d\n", file, line)
		fmt.Printf("Children: %v\n", node.Children())
		fmt.Println("---")
	}

	node := terr.TraceTree(newErr)
	printNode(node)
	printNode(node.Children()[0])
	printNode(node.Children()[1])
}
