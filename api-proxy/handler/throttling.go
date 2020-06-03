package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
	"unsafe"

	log "github.com/cihub/seelog"

	"github.com/hailo-platform/H2O/api-proxy/errors"
	"github.com/hailo-platform/H2O/api-proxy/session"
)

const (
	// How often the synchroniser checks-in with the API throttling service
	// @TODO: Should this come from config?
	synchronisationInterval = 5 * time.Second
	defaultBufferSize       = 5000
)

type (
	throttledBucketsT map[string]bool
	bucketBufferT     map[string]*uint64
)

// A ThrottlingHandler is a decorator around another HTTP handler that implements our API throttling behaviour. It
// buckets inbound requests, records statistics about request volume to each bucket, and throttles full buckets. If a
// request is throttled, it will proceed no further up the handler chain.
type ThrottlingHandler struct {
	Handler http.Handler
	// This is seriously nasty, but we need atomic pointer operations here, so we need unsafe pointers.
	// /me dies inside
	throttledBuckets unsafe.Pointer // *throttledBucketsT: buckets to throttle
	bucketBuffer     unsafe.Pointer // *bucketBufferT: inbound per-bucket request count buffer
	ingesterChan     chan string    // inbound requests to be added to the buffer
	srv              *HailoServer
}

func NewThrottlingHandler(h http.Handler, srv *HailoServer) *ThrottlingHandler {
	throttled := make(throttledBucketsT)
	buf := make(bucketBufferT, defaultBufferSize)

	t := &ThrottlingHandler{
		Handler:          h,
		throttledBuckets: (unsafe.Pointer)(&throttled),
		bucketBuffer:     (unsafe.Pointer)(&buf),
		ingesterChan:     make(chan string, 500000),
		srv:              srv,
	}
	srv.Tomb.Go(t.ingesterWorker)
	srv.Tomb.Go(t.synchroniser)
	return t
}

// ingesterWorker takes bucket names from the ingesterChan and increments the appropriate record in the bucketBuffer
func (t *ThrottlingHandler) ingesterWorker() error {
	// Note that if ingesterWorker() is mid-flight incrementing a value in the buffer when synchroniser() swaps the
	// buffer for a new one, the single increment will be lost. This could be mitigated by comparing the pointer again
	// at the end of the work loop, but it's really not a big deal for one request to fall through the cracks.
	for {
		select {
		case <-t.srv.Tomb.Dying():
			// Die in response to tomb death
			log.Tracef("[Throttler:ingesterWorker] Dying in response to server tomb death")
			return nil
		case bucketKey, ok := <-t.ingesterChan:
			if !ok {
				// Die in response to channel closure
				log.Tracef("[Throttler:ingesterWorker] Dying in response to channel closure")
				return nil
			}

			loadedP := atomic.LoadPointer(&t.bucketBuffer)
			buf := *(*bucketBufferT)(loadedP)

			if valPtr, ok := buf[bucketKey]; !ok {
				// As adding a value to a map is non-atomic, we have no choice but to copy the map and add the new key,
				// retrying if the map was switched under us while this was in-process
				// @TODO: Replace with OB's concurrent hash table impl
				for {
					newBuf := make(bucketBufferT, len(buf)+1)
					for k, v := range buf {
						newBuf[k] = v
					}
					_one := uint64(1)
					newBuf[bucketKey] = &_one

					if atomic.CompareAndSwapPointer(&t.bucketBuffer, loadedP, (unsafe.Pointer)(&newBuf)) {
						// CAS was uncontended; we have successfully added the value to the map
						break
					}

					// CAS was contended; go again with a reloaded pointer
					loadedP = atomic.LoadPointer(&t.bucketBuffer)
					buf = *(*bucketBufferT)(loadedP)
				}
			} else {
				atomic.AddUint64(valPtr, uint64(1))
			}
		}
	}
}

// synchroniser periodically sends the bucketBuffer to the API throttling service, and updates throttledBuckets
// accordingly. When the bucketBuffer is sent, it is replaced (atomically) with a new buffer.
func (t *ThrottlingHandler) synchroniser() error {
	tick := time.NewTicker(synchronisationInterval)
	defer tick.Stop()

	for {
		select {
		case <-t.srv.Tomb.Dying():
			// Die in response to tomb death
			log.Tracef("[Throttler:synchroniser] Dying in response to server tomb death")
			return nil
		case _, ok := <-tick.C:
			if !ok {
				// Die in response to channel closure
				log.Tracef("[Throttler:synchroniser] Dying in response to channel closure")
				return nil
			}

			// Swap the buffer for a shiny new one
			newBuf := make(bucketBufferT, defaultBufferSize)
			bufP := (*bucketBufferT)(atomic.SwapPointer(&t.bucketBuffer, (unsafe.Pointer)(&newBuf)))

			// DO NOT bail here; we still need to retrieve buckets to be throttled even if no increments to report
			log.Debugf("[Throttler:synchroniser] Reporting %d increments", len(*bufP))
			start := time.Now()
			tbP, err := reportIncrements(bufP)
			var tb throttledBucketsT
			if err != nil {
				log.Errorf("[Throttler:synchroniser] Failed to report increments in %s: %s", time.Since(start).String(),
					err.Error())
				// Reset the throttled bucket list (don't throttle anything in the failure case)
				tb = make(throttledBucketsT, 0)
			} else {
				log.Debugf("[Throttler:synchroniser] Successfully reported increments in %s",
					time.Since(start).String())
				tb = *tbP
				log.Debugf("[Throttler:synchroniser] Got %d buckets to throttle", len(tb))
			}

			atomic.StorePointer(&(t.throttledBuckets), (unsafe.Pointer)(&tb))
		}
	}
}

// buckets returns the buckets that this request falls into
func (t *ThrottlingHandler) buckets(r *http.Request) []string {
	result := make([]string, 0, 1)

	// Session ID
	if sessId := session.SessionId(r); sessId != "" {
		result = append(result, fmt.Sprintf("sessId:%s", sessId))
	}

	return result
}

// anyThrottled checks if any of the passed buckets are to be throttled
func (t *ThrottlingHandler) anyThrottled(bucks []string) bool {
	throttledP := (*throttledBucketsT)(atomic.LoadPointer(&(t.throttledBuckets)))
	if throttledP == nil {
		return false
	}
	throttled := *throttledP

	for _, b := range bucks {
		if _, ok := throttled[b]; ok {
			return true
		}
	}
	return false
}

func (t *ThrottlingHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// Record increments to the relevant buckets
	bucks := t.buckets(r)
	for _, b := range bucks {
		select {
		case t.ingesterChan <- b:
		default:
			log.Warn("[Throttler:ServeHTTP] Could not add bucket to ingestion buffer")
			break
		}
	}

	if t.anyThrottled(bucks) {
		// The request should be throttled
		rw.Header().Set("Content-Type", "application/json; charset=utf-8")
		rw.WriteHeader(429)

		b, marshalErr := json.Marshal(errors.ErrorBody{
			Status:     false,
			Payload:    "Client error: rate limit exceeded",
			Number:     429,
			DottedCode: "com.hailo-platform/H2O.api.throttled",
			Context:    nil,
		})
		if marshalErr != nil {
			log.Warn("[Throttler:ServerHTTP] Error marshalling error")
			return
		}

		rw.Write(b)
		return
	}

	t.Handler.ServeHTTP(rw, r)
}
