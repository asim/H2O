package trace_test

import (
	"bytes"
	"github.com/hailo-platform/H2O/api-proxy/trace"
	"github.com/hailo-platform/H2O/service/config"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestTraceEnabling(t *testing.T) {
	// Tracing should not happen with no X-H-TRACEID, and a pcChance of 0
	buf := bytes.NewBufferString(`{"hailo":{"api":{"trace":{"pcChance":0.0}}}}`)
	origConfigBuf := bytes.NewBuffer(config.Raw())
	config.Load(buf)
	defer config.Load(origConfigBuf)

	req, _ := http.NewRequest("GET", "/", nil)
	traceInfo := trace.Start(req)
	assert.Equal(t, "", traceInfo.TraceId, "TraceId should be blank with no X-H-TRACEID header and pcChance=0")

	// It should happen with X-H-TRACE set to 1 (in this case, a random trace ID will be assigned)
	req.Header.Set("X-H-TRACE", "1")
	traceInfo = trace.Start(req)
	assert.NotEqual(t, traceInfo.TraceId, "", "TraceId should not be blank with X-H-TRACE=1")
	assert.NotEqual(t, traceInfo.TraceId, "1", "TraceId should be assigned a new ID with X-H-TRACE=1")
	assert.True(t, traceInfo.PersistentTrace, "PersistentTrace should be true with a user-initiated trace")

	// It should happen with X-H-TRACEID set to an actual ID (and in this case, the trace ID should carry through)
	req.Header.Set("X-H-TRACEID", "xxx-trace-id-test-xxx")
	traceInfo = trace.Start(req)
	assert.NotEqual(t, traceInfo.TraceId, "", "TraceId should not be blank with X-H-TRACEID specified")
	assert.Equal(t, traceInfo.TraceId, "xxx-trace-id-test-xxx", "TraceId should carry though from X-H-TRACEID")
	assert.True(t, traceInfo.PersistentTrace, "PersistentTrace should be true with a user-initiated trace")

	// In the case of a randomly-chosen request being traced, persistence should be disabled
	req.Header.Del("X-H-TRACE")
	req.Header.Del("X-H-TRACEID")
	buf = bytes.NewBufferString(`{"hailo":{"api":{"trace":{"pcChance":1.0}}}}`)
	config.Load(buf)
	traceInfo = trace.Start(req)
	assert.NotEqual(t, traceInfo.TraceId, "", "Random tracing should always be enabled with pcChance=0")
	assert.False(t, traceInfo.PersistentTrace, "PersistentTrace should be false with a randomly-initiated trace")
}
