package trace

import (
	"math/rand"
	"net/http"
	"time"

	traceproto "github.com/hailo-platform/H2O/platform/proto/trace"
	"github.com/hailo-platform/H2O/service/config"
	strace "github.com/hailo-platform/H2O/service/trace"
	"github.com/hailo-platform/H2O/protobuf/proto"
	gouuid "github.com/nu7hatch/gouuid"
)

type APITraceInfo struct {
	// The ID given to the trace. If this is a blank string, tracing is not enabled
	TraceId         string
	PersistentTrace bool
}

// Start will decide if we should trace a request, and if so, trigger a "start" event, which indicates we have kicked
// off a trace at our borders (the API layer)
func Start(r *http.Request) *APITraceInfo {
	traceId := r.Header.Get("X-H-TRACEID")
	userInitiatedTrace := r.Header.Get("X-H-TRACE") == "1" || traceId != ""
	if userInitiatedTrace || randomTrace() {
		if traceId == "" {
			u4, _ := gouuid.NewV4()
			traceId = u4.String()
		}
	}

	if traceId != "" {
		strace.Send(&traceproto.Event{
			Timestamp:       proto.Int64(int64(time.Now().UnixNano())),
			TraceId:         proto.String(traceId),
			Type:            traceproto.Event_START.Enum(),
			PersistentTrace: proto.Bool(userInitiatedTrace),
		})
	}

	return &APITraceInfo{
		TraceId:         traceId,
		PersistentTrace: userInitiatedTrace,
	}
}

// Write makes a decision as to what we should attach to the output based on trace
func Write(rw http.ResponseWriter, traceInfo *APITraceInfo) {
	if traceInfo.TraceId != "" {
		rw.Header().Set("X-H-TRACEID", traceInfo.TraceId)
	}
}

// randomTrace determines, using configured pcChance (0.0 -> 1.0), whether we should initiate trace for a request
func randomTrace() bool {
	pcChance := config.AtPath("hailo", "api", "trace", "pcChance").AsFloat64(0)
	if pcChance <= 0 {
		return false
	} else {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		return (r.Float64() < pcChance)
	}
}
