package multierror

import (
	"fmt"
	"testing"
)

func TestMultiError(t *testing.T) {
	testCases := []struct {
		c   int
		err string
	}{
		{0, "No errors"},
		{1, "Something went wrong 1"},
		{2, "Something went wrong 1, and 1 more error"},
		{5, "Something went wrong 1, and 4 more errors"},
	}

	for _, tc := range testCases {
		me := New()

		for i := 1; i <= tc.c; i++ {
			me.Add(fmt.Errorf("Something went wrong %d", i))
		}

		if me.Error() != tc.err {
			t.Errorf(`Incorrect response for %d errors, got "%s" by expected "%s"`, tc.c, me.Error(), tc.err)
		}
	}
}
