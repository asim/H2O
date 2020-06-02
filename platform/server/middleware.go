package server

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/protobuf/proto"

	errors "github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/stats"
	inst "github.com/HailoOSS/service/instrumentation"
	trace "github.com/HailoOSS/service/trace"

	traceproto "github.com/HailoOSS/platform/proto/trace"
)

// commonLogHandler will log to w using the Apache common log format
// http://httpd.apache.org/docs/2.2/logs.html#common
// If w is nil, nothing will be logged
func commonLoggerMiddleware(w io.Writer) Middleware {
	return func(ep *Endpoint, h Handler) Handler {
		// If no writer is passed to middleware just return the handler
		if w == nil {
			return h
		}

		return func(req *Request) (proto.Message, errors.Error) {
			var userId string
			if req.Auth() != nil && req.Auth().AuthUser() != nil {
				userId = req.Auth().AuthUser().Id
			}

			var err errors.Error
			var m proto.Message

			// In defer in case the handler panics
			defer func() {
				status := uint32(200)
				if err != nil {
					status = err.HttpCode()
				}

				size := 0
				if m != nil {
					log.Debug(m.String())
					size = len(m.String())
				}

				fmt.Fprintf(w, "%s - %s [%s] \"%s %s %s\" %d %d\n",
					req.From(),
					userId,
					time.Now().Format("02/Jan/2006:15:04:05 -0700"),
					"GET", // Treat them all as GET's at the moment
					req.Endpoint(),
					"HTTP/1.0", // Has to be HTTP or apachetop ignores it
					status,
					size,
				)
			}()

			// Execute the actual handler
			m, err = h(req)
			return m, err
		}
	}
}

// tokenConstrainedMiddleware limits the max concurrent requests handled per caller
func tokenConstrainedMiddleware(ep *Endpoint, h Handler) Handler {
	return func(req *Request) (proto.Message, errors.Error) {
		callerName := req.From()
		if callerName == "" {
			callerName = "unknown"
		}
		tokenBucketName := fmt.Sprintf("server.tokens.%s", callerName)
		reqsBucketName := fmt.Sprintf("server.inflightrequests.%s", callerName)
		tokC := tokensChan(callerName)

		select {
		case t := <-tokC:
			defer func() {
				atomic.AddUint64(&inFlightRequests, ^uint64(0)) // This is actually a subtraction
				tokC <- t                                       // Return the token to the pool
			}()

			nowInFlight := atomic.AddUint64(&inFlightRequests, 1) // Update active request counters
			inst.Gauge(1.0, tokenBucketName, len(tokC))
			inst.Gauge(1.0, reqsBucketName, int(nowInFlight))
			return h(req)
		case <-time.After(time.Duration(ep.Mean) * time.Millisecond):
			inst.Gauge(1.0, tokenBucketName, len(tokC))
			inst.Counter(1.0, "server.error.capacity", 1)

			return nil, errors.InternalServerError("com.HailoOSS.kernel.server.capacity",
				fmt.Sprintf("Server %v out of capacity", Name))
		}
	}
}

// instrumentedHandler wraps the handler to provide instrumentation
func instrumentedMiddleware(ep *Endpoint, h Handler) Handler {
	return func(req *Request) (rsp proto.Message, err errors.Error) {
		start := time.Now()
		// In a defer in case the handler panics
		defer func() {
			stats.Record(ep, err, time.Since(start))
			if err == nil {
				inst.Timing(1.0, "success."+ep.Name, time.Since(start))
				return
			}
			inst.Counter(1.0, fmt.Sprintf("server.error.%s", err.Code()), 1)
			switch err.Type() {
			case errors.ErrorBadRequest, errors.ErrorNotFound:
				// Ignore errors that are caused by clients
				// TODO: consider a new stat for clienterror?
				inst.Timing(1.0, "success."+ep.Name, time.Since(start))
				return
			default:
				inst.Timing(1.0, "error."+ep.Name, time.Since(start))
			}
		}()
		rsp, err = h(req)
		return rsp, err
	}
}

// tracingMiddleware adds tracing to a handler
func tracingMiddleware(ep *Endpoint, h Handler) Handler {
	return func(req *Request) (rsp proto.Message, err errors.Error) {
		start := time.Now()
		traceIn(req)
		defer traceOut(req, rsp, err, time.Since(start))
		rsp, err = h(req)
		return rsp, err
	}
}

// authMiddleware only calls the handler is the auth check passes
func authMiddleware(ep *Endpoint, h Handler) Handler {
	return func(req *Request) (proto.Message, errors.Error) {
		if err := ep.Authoriser.Authorise(req); err != nil {
			return nil, err
		}
		req.Auth().SetAuthorised(true)
		return h(req)
	}
}

func waitGroupMiddleware(ep *Endpoint, h Handler) Handler {
	return func(req *Request) (proto.Message, errors.Error) {
		requestsWg.Add(1)
		defer requestsWg.Done()

		return h(req)
	}
}

// traceIn traces a request inbound to a service to handle
func traceIn(req *Request) {
	if req.shouldTrace() {
		go trace.Send(&traceproto.Event{
			Timestamp:         proto.Int64(time.Now().UnixNano()),
			TraceId:           proto.String(req.TraceID()),
			Type:              traceproto.Event_IN.Enum(),
			MessageId:         proto.String(req.MessageID()),
			ParentMessageId:   proto.String(req.ParentMessageID()),
			From:              proto.String(req.From()),
			To:                proto.String(fmt.Sprintf("%v.%v", req.Service(), req.Endpoint())),
			Hostname:          proto.String(hostname),
			Az:                proto.String(az),
			Payload:           proto.String(""), // @todo
			HandlerInstanceId: proto.String(InstanceID),
			PersistentTrace:   proto.Bool(req.TraceShouldPersist()),
		})
	}
}

// traceOut traces a request outbound from a service handler
func traceOut(req *Request, msg proto.Message, err errors.Error, d time.Duration) {
	if req.shouldTrace() {
		e := &traceproto.Event{
			Timestamp:         proto.Int64(time.Now().UnixNano()),
			TraceId:           proto.String(req.TraceID()),
			Type:              traceproto.Event_OUT.Enum(),
			MessageId:         proto.String(req.MessageID()),
			ParentMessageId:   proto.String(req.ParentMessageID()),
			From:              proto.String(req.From()),
			To:                proto.String(fmt.Sprintf("%v.%v", req.Service(), req.Endpoint())),
			Hostname:          proto.String(hostname),
			Az:                proto.String(az),
			Payload:           proto.String(""), // @todo
			HandlerInstanceId: proto.String(InstanceID),
			Duration:          proto.Int64(int64(d)),
			PersistentTrace:   proto.Bool(req.TraceShouldPersist()),
		}
		if err != nil {
			e.ErrorCode = proto.String(err.Code())
			e.ErrorDescription = proto.String(err.Description())
		}
		go trace.Send(e)
	}
}
