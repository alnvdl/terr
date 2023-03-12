package terr_test

import (
	"fmt"
	"runtime"

	"github.com/alnvdl/terr"
)

type ValidationError struct{ msg string }

func (e *ValidationError) Error() string {
	return e.msg
}

func NewValidationError(msg string) error {
	// Custom errors can define a location for the error. In this case, we set
	// it to the location of the caller of this function.
	_, file, line, _ := runtime.Caller(1)
	return terr.TraceWithLocation(&ValidationError{msg}, file, line)
}

// This example shows how to adding tracing information to custom error types
// using TraceWithLocation.
func ExampleTraceWithLocation() {
	// err will be annotated with the line number of the next line.
	err := NewValidationError("x must be >= 0")
	fmt.Printf("%@\n", err)
}
