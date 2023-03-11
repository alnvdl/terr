package terr_test

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/alnvdl/terr"
)

func getLocation(depth int) (string, int) {
	_, file, line, _ := runtime.Caller(depth + 1)
	return file, line
}

func assertString(t *testing.T, got, want string) {
	if got != want {
		t.Fatalf("want %q got %q", want, got)
	}
}

func assertBool(t *testing.T, got, want bool) {
	if got != want {
		t.Fatalf("want %t got %t", want, got)
	}
}

func assertErrorIsNil(t *testing.T, got error) {
	if got != nil {
		t.Fatalf("want nil error, got %#v", got)
	}
}

func assertTraceTreeEquals(t *testing.T, got terr.TracedError, want terr.TracedError) {
	if got == nil && want == nil {
		return
	}
	if got != nil && want == nil {
		t.Fatalf("want traced error children is nil but got traced error children isn't nil")
	}
	if got == nil && want != nil {
		t.Fatalf("want trace is not nil but got trace is nil")
	}

	assertString(t, got.Location(), want.Location())
	assertString(t, got.Error(), want.Error())
	if len(got.Location()) != len(want.Location()) {
		t.Fatalf("want trace tree with %d children, got trace tree with %d children: want %#v got %#v ", len(want.Children()), len(got.Children()), want.Children(), got.Children())
	}
	for i := range got.Children() {
		assertTraceTreeEquals(t, got.Children()[i], want.Children()[i])
	}
}

func TestTrace(t *testing.T) {
	file, line := getLocation(0)
	err := terr.Newf("fail")
	tracedErr := terr.Trace(err)

	assertString(t, tracedErr.Error(), "fail")
	assertBool(t, errors.Is(tracedErr, err), true)
	// tracedErr.Unwrap() should still return nil, because no wrapping took place.
	assertErrorIsNil(t, errors.Unwrap(tracedErr))
	assertString(t, fmt.Sprintf("%@", tracedErr), strings.Join([]string{
		fmt.Sprintf("fail @ %s:%d", file, line+2),
		fmt.Sprintf("\tfail @ %s:%d", file, line+1),
	}, "\n"))
}

type customError struct {
	value string
}

func (e *customError) Error() string {
	return "base"
}

func TestNewf(t *testing.T) {
	base := &customError{value: "test"}
	file, line := getLocation(0)
	err := terr.Newf("fail: %w", base)
	tracedErr := terr.Trace(err)
	wrappedErr := terr.Newf("wrapped: %w", tracedErr)
	maskedErr := terr.Newf("masked: %v", wrappedErr)
	f := terr.Newf("newf: %w", maskedErr)

	assertString(t, f.Error(), "newf: masked: wrapped: fail: base")

	assertBool(t, errors.Is(f, maskedErr), true)
	assertBool(t, errors.Is(f, wrappedErr), false)

	var ce *customError
	ok := errors.As(wrappedErr, &ce)
	assertBool(t, ok, true)
	assertString(t, ce.value, "test")

	unwrapped := errors.Unwrap(wrappedErr)
	assertBool(t, unwrapped == tracedErr, true)

	assertString(t, fmt.Sprintf("%@", f), strings.Join([]string{
		fmt.Sprintf("newf: masked: wrapped: fail: base @ %s:%d", file, line+5),
		fmt.Sprintf("\tmasked: wrapped: fail: base @ %s:%d", file, line+4),
		fmt.Sprintf("\t\twrapped: fail: base @ %s:%d", file, line+3),
		fmt.Sprintf("\t\t\tfail: base @ %s:%d", file, line+2),
		fmt.Sprintf("\t\t\t\tfail: base @ %s:%d", file, line+1),
	}, "\n"))
}

type traceTreeNode struct {
	err      string
	location string
	children []*traceTreeNode
}

func (t *traceTreeNode) Error() string {
	return t.err
}

func (t *traceTreeNode) Location() string {
	return t.location
}

func (t *traceTreeNode) Children() []terr.TracedError {
	terrs := make([]terr.TracedError, len(t.children))
	for i := range t.children {
		terrs[i] = t.children[i]
	}
	return terrs
}

var _ terr.TracedError = (*traceTreeNode)(nil)

func TestNewfMultiple(t *testing.T) {
	file, line := getLocation(0)
	err1 := terr.Newf("fail")
	terr1 := terr.Trace(err1)
	err2 := terr.Newf("problem")
	terr2 := terr.Newf("wrapped: %w", err2)
	f := terr.Newf("errors: %w and %v", terr1, terr2)

	assertString(t, f.Error(), "errors: fail and wrapped: problem")
	assertBool(t, errors.Is(f, terr1), true)  // %w was used.
	assertBool(t, errors.Is(f, terr2), false) // %v was used

	assertString(t, fmt.Sprintf("%@", f), strings.Join([]string{
		fmt.Sprintf("errors: fail and wrapped: problem @ %s:%d", file, line+5),
		fmt.Sprintf("\tfail @ %s:%d", file, line+2),
		fmt.Sprintf("\t\tfail @ %s:%d", file, line+1),
		fmt.Sprintf("\twrapped: problem @ %s:%d", file, line+4),
		fmt.Sprintf("\t\tproblem @ %s:%d", file, line+3),
	}, "\n"))

	// TraceTree has information about the whole tree, but it's not represented
	// as a string.
	assertTraceTreeEquals(t, terr.TraceTree(f), &traceTreeNode{
		err:      f.Error(),
		location: fmt.Sprintf("%s:%d", file, line+5),
		children: []*traceTreeNode{{
			err:      terr1.Error(),
			location: fmt.Sprintf("%s:%d", file, line+2),
			children: []*traceTreeNode{{
				err:      err1.Error(),
				location: fmt.Sprintf("%s:%d", file, line+1),
			}},
		}, {
			err:      terr2.Error(),
			location: fmt.Sprintf("%s:%d", file, line+4),
			children: []*traceTreeNode{{
				err:      err2.Error(),
				location: fmt.Sprintf("%s:%d", file, line+3),
			}},
		},
		},
	})
}

func TestNil(t *testing.T) {
	assertErrorIsNil(t, terr.Trace(nil))
	assertTraceTreeEquals(t, terr.TraceTree(nil), nil)
}
