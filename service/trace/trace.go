// Trace package facilitates the recording of trace events which allow us to follow
// a request right the way through the H2 platform
package trace

import (
	"fmt"
	log "github.com/cihub/seelog"
	traceproto "github.com/HailoOSS/platform/proto/trace"
	"github.com/HailoOSS/protobuf/proto"
)

// Send will ping off a trace event
func Send(e *traceproto.Event) error {

	// Marshal the trace here, so it's done concurrently
	msg, err := proto.Marshal(e)
	if err != nil {
		return err
	}

	// Send the marshaled trace, dropping if the backend is at capacity
	select {
	case traceChan <- msg:
		log.Tracef("Enqueued trace event message %v", e)
	default:
		// Channel is full, dropping trace :(
		// @todo We could instrument this, but if we're dropping traces we probably have serious problems?
		return fmt.Errorf("Dropping trace as channel is full")
	}

	return nil
}
