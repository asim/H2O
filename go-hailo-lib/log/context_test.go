// +build !race

// Note, this test fails under the race detector because of the way the detector rewrites the application
// source code. The above build tag will exclude this test from the race detector, but not from normal tests

package log

import (
	"bytes"
	"strings"
	"testing"

	seelog "github.com/cihub/seelog"
)

// Ensures that the log output contains the correct filename
func TestFilename(t *testing.T) {
	var buf bytes.Buffer

	logger, _ := seelog.LoggerFromWriterWithMinLevelAndFormat(&buf, seelog.InfoLvl, "%File")
	seelog.ReplaceLogger(logger)

	c := NewContext("")

	c.Info("foo")

	msg := buf.String()
	if !strings.Contains(msg, "context_test.go") {
		t.Errorf("Expecting log message to contain correct filename, but got: %s", msg)
	}
}
