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

func assertEquals[T comparable](t *testing.T, got, want T) {
	if got != want {
		t.Fatalf("want %#v got %#v", want, got)
	}
}

func assertErrorIsNil(t *testing.T, got error) {
	if got != nil {
		t.Fatalf("want nil error, got %#v", got)
	}
}

func assertTraceTreeEquals(t *testing.T, got terr.ErrorTracer, want terr.ErrorTracer) {
	if got == nil && want == nil {
		return
	}
	if got != nil && want == nil {
		t.Fatalf("want traced error children is nil but got traced error children isn't nil")
	}
	if got == nil && want != nil {
		t.Fatalf("want trace is not nil but got trace is nil")
	}

	gotFile, gotLine := got.Location()
	wantFile, wantLine := want.Location()
	assertEquals(t, gotFile, wantFile)
	assertEquals(t, gotLine, wantLine)
	assertEquals(t, got.Error(), want.Error())
	if len(got.Children()) != len(want.Children()) {
		t.Fatalf("want trace tree with %d children, got trace tree with %d children: want %#v got %#v ",
			len(want.Children()),
			len(got.Children()),
			want.Children(),
			got.Children())
	}
	for i := range got.Children() {
		assertTraceTreeEquals(t, got.Children()[i], want.Children()[i])
	}
}

func TestTrace(t *testing.T) {
	file, line := getLocation(0)
	err := terr.Newf("fail")
	tracedErr := terr.Trace(err)

	assertEquals(t, tracedErr.Error(), "fail")
	assertEquals(t, errors.Is(tracedErr, err), true)
	// tracedErr.Unwrap() should still return nil, because no wrapping took place.
	assertErrorIsNil(t, errors.Unwrap(tracedErr))
	assertEquals(t, fmt.Sprintf("%@", tracedErr), strings.Join([]string{
		fmt.Sprintf("fail @ %s:%d", file, line+2),
		fmt.Sprintf("\tfail @ %s:%d", file, line+1),
	}, "\n"))

	tracedErrOpts := terr.Trace(err,
		terr.WithLocation("somefile.go", 123),
		terr.WithChildren([]error{tracedErr}),
	)
	assertEquals(t, tracedErrOpts.Error(), "fail")
	assertEquals(t, errors.Is(tracedErrOpts, err), true)
	// tracedErr.Unwrap() should still return nil, because no wrapping took place.
	assertErrorIsNil(t, errors.Unwrap(tracedErrOpts))
	assertEquals(t, fmt.Sprintf("%@", tracedErrOpts), strings.Join([]string{
		fmt.Sprintf("fail @ %s:%d", "somefile.go", 123),
		fmt.Sprintf("\tfail @ %s:%d", file, line+1),
		// tracedErrOpts included tracedErr as a child.
		fmt.Sprintf("\tfail @ %s:%d", file, line+2),
		fmt.Sprintf("\t\tfail @ %s:%d", file, line+1),
	}, "\n"))
}

type customError struct {
	value string
}

func (e *customError) Error() string {
	return e.value
}

func TestNewf(t *testing.T) {
	base := &customError{value: "test"}
	file, line := getLocation(0)
	err := terr.Newf("fail: %w", base)
	tracedErr := terr.Trace(err)
	wrappedErr := terr.Newf("wrapped: %w", tracedErr)
	maskedErr := terr.Newf("masked: %v", wrappedErr)
	wrappedAgain := terr.Newf("newf: %w", maskedErr)

	assertEquals(t, wrappedAgain.Error(), "newf: masked: wrapped: fail: test")

	assertEquals(t, errors.Is(wrappedAgain, maskedErr), true)
	assertEquals(t, errors.Is(wrappedAgain, wrappedErr), false)

	var ce *customError
	ok := errors.As(wrappedErr, &ce)
	assertEquals(t, ok, true)
	assertEquals(t, ce.value, "test")

	unwrapped := errors.Unwrap(wrappedErr)
	assertEquals(t, unwrapped == tracedErr, true)

	assertEquals(t, fmt.Sprintf("%@", wrappedAgain), strings.Join([]string{
		fmt.Sprintf("newf: masked: wrapped: fail: test @ %s:%d", file, line+5),
		fmt.Sprintf("\tmasked: wrapped: fail: test @ %s:%d", file, line+4),
		fmt.Sprintf("\t\twrapped: fail: test @ %s:%d", file, line+3),
		fmt.Sprintf("\t\t\tfail: test @ %s:%d", file, line+2),
		fmt.Sprintf("\t\t\t\tfail: test @ %s:%d", file, line+1),
	}, "\n"))
}

type traceTreeNode struct {
	err      string
	file     string
	line     int
	children []*traceTreeNode
}

func (t *traceTreeNode) Error() string {
	return t.err
}

func (t *traceTreeNode) Location() (string, int) {
	return t.file, t.line
}

func (t *traceTreeNode) Children() []terr.ErrorTracer {
	terrs := make([]terr.ErrorTracer, len(t.children))
	for i := range t.children {
		terrs[i] = t.children[i]
	}
	return terrs
}

var _ terr.ErrorTracer = (*traceTreeNode)(nil)

func TestNewfMultiple(t *testing.T) {
	file, line := getLocation(0)
	err1 := terr.Newf("fail")
	terr1 := terr.Trace(err1)
	err2 := terr.Newf("problem")
	terr2 := terr.Newf("wrapped: %w", err2)
	f := terr.Newf("errors: %w and %v", terr1, terr2)

	assertEquals(t, f.Error(), "errors: fail and wrapped: problem")
	assertEquals(t, errors.Is(f, terr1), true)  // %w was used.
	assertEquals(t, errors.Is(f, terr2), false) // %v was used

	assertEquals(t, fmt.Sprintf("%@", f), strings.Join([]string{
		fmt.Sprintf("errors: fail and wrapped: problem @ %s:%d", file, line+5),
		fmt.Sprintf("\tfail @ %s:%d", file, line+2),
		fmt.Sprintf("\t\tfail @ %s:%d", file, line+1),
		fmt.Sprintf("\twrapped: problem @ %s:%d", file, line+4),
		fmt.Sprintf("\t\tproblem @ %s:%d", file, line+3),
	}, "\n"))

	// TraceTree has information about the whole tree, but it's not represented
	// as a string.
	assertTraceTreeEquals(t, terr.TraceTree(f), &traceTreeNode{
		err:  f.Error(),
		file: file,
		line: line + 5,
		children: []*traceTreeNode{{
			err:  terr1.Error(),
			file: file,
			line: line + 2,
			children: []*traceTreeNode{{
				err:  err1.Error(),
				file: file,
				line: line + 1,
			}},
		}, {
			err:  terr2.Error(),
			file: file,
			line: line + 4,
			children: []*traceTreeNode{{
				err:  err2.Error(),
				file: file,
				line: line + 3,
			}},
		},
		},
	})
}

func TestNil(t *testing.T) {
	assertErrorIsNil(t, terr.Trace(nil))
	assertErrorIsNil(t, terr.Trace(nil, terr.WithLocation("somefile.go", 123)))

	assertTraceTreeEquals(t, terr.TraceTree(nil), nil)
}
