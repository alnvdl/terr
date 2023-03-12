package terr_test

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/alnvdl/terr"
)

// This example shows how to combine different terr functions and print a
// traced error tree at the end.
func Example() {
	err := terr.Newf("base")
	traced := terr.Trace(err)
	wrapped := terr.Newf("wrapped: %w", traced)
	masked := terr.Newf("masked: %v", wrapped)
	fmt.Printf("%@\n", masked)
}

// This example shows how Newf interacts with a non-traced error compared to
// when it receives a traced error. Traced errors are included in the trace
// regardless of the fmt verb used for them, while non-traced errors are
// handled as usual, but do not get included in the trace.
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

// This example shows how terr.Trace interacts with a non-traced error compared
// to when it receives a traced error.
func ExampleTrace() {
	nonTracedErr := errors.New("non-traced")
	fmt.Printf("%@\n", terr.Trace(nonTracedErr))
	fmt.Println("---")
	tracedErr := terr.Newf("traced")
	fmt.Printf("%@\n", terr.Trace(tracedErr))
}

type ValidationError struct{ msg string }

func (e *ValidationError) Error() string {
	return e.msg
}

func NewValidationError(msg string) error {
	_, file, line, _ := runtime.Caller(1)
	return terr.TraceWithLocation(&ValidationError{msg}, file, line)
}

// This example shows how to adding tracing information to custom error types
// using TraceWithLocation. Custom errors constructors like NewValidationError
// can define a location for the errors they return. In this case, that
// location is being set it to the location of the NewValidationError caller.
func ExampleTraceWithLocation() {
	// err will be annotated with the line number of the following line.
	err := NewValidationError("x must be >= 0")
	fmt.Printf("%@\n", err)
	fmt.Println("---")

	// error.As works.
	var customErr *ValidationError
	ok := errors.As(err, &customErr)
	fmt.Println("Is ValidationError:", ok)
	fmt.Println("Custom error message:", customErr.msg)
}

// This example shows how to use the n-ary traced error tree returned by
// terr.TraceTree.
func ExampleTraceTree() {
	nonTracedErr := errors.New("non-traced")
	tracedErr1 := terr.Newf("traced 1")
	tracedErr2 := terr.Newf("traced 2")
	newErr := terr.Newf("%w, %v, %w",
		nonTracedErr,
		tracedErr1,
		tracedErr2)

	printNode := func(node terr.TracedError) {
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
