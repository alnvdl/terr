package terr_test

import (
	"fmt"

	"github.com/alnvdl/terr"
)

// This example shows how to call different terr functions and print a traced
// error tree at the end.
func Example() {
	err := terr.Newf("base")
	traced := terr.Trace(err)
	wrapped := terr.Newf("wrapped: %w", traced)
	masked := terr.Newf("masked: %v", wrapped)
	fmt.Printf("%@\n", masked)
}
