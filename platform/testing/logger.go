package testing

import (
	"strings"
	"testing"

	log "github.com/cihub/seelog"
)

type testLogger struct {
	tb testing.TB
}

func (t *testLogger) ReceiveMessage(message string, level log.LogLevel, context log.LogContextInterface) error {
	if t != nil && t.tb != nil {
		t.tb.Log(strings.TrimSpace(message))
	}

	return nil
}

func (t *testLogger) AfterParse(initArgs log.CustomReceiverInitArgs) error {
	return nil
}

func (t *testLogger) Flush() {
}

func (t *testLogger) Close() error {
	return nil
}
