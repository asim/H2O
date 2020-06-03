package handler

import (
	"errors"
	"sync/atomic"

	"github.com/hailo-platform/H2O/protobuf/proto"

	checkinproto "github.com/hailo-platform/H2O/api-throttling-service/proto/checkin"
	"github.com/hailo-platform/H2O/platform/multiclient"
	"github.com/hailo-platform/H2O/service/config"
)

// reportIncrements sends recently-recorded bucket increments to the API throttling service, and in response returns a
// buckets that should be throttled
func reportIncrements(bufP *bucketBufferT) (*throttledBucketsT, error) {
	result := make(throttledBucketsT, 0)

	// If increment reporting is disabled, return immediately
	if !config.AtPath("hailo", "service", "api", "throttling", "reportIncrements").AsBool() {
		return &result, nil
	}

	buf := *bufP
	bucketReqs := make([]*checkinproto.BucketRequests, 0, len(buf))
	for k, valPtr := range buf {
		bucketReqs = append(bucketReqs, &checkinproto.BucketRequests{
			BucketKey:    proto.String(k),
			RequestCount: proto.Uint64(atomic.LoadUint64(valPtr)),
		})
	}

	rsp := new(checkinproto.Response)
	req := multiclient.New().AddScopedReq(&multiclient.ScopedReq{
		Uid:      "increments",
		From:     newthrottleSyncScoper(),
		Service:  "com.hailocab.service.api-throttling",
		Endpoint: "checkin",
		Req: &checkinproto.Request{
			BucketRequests: bucketReqs,
		},
		Rsp: rsp,
	}).Execute()

	if req.AnyErrors() {
		return nil, errors.New(req.PlatformError(".servicecall.reportincrements").Error())
	}

	throttledBucketsRsp := rsp.GetThrottledBuckets()
	for _, bucketKey := range throttledBucketsRsp {
		result[bucketKey] = true
	}

	return &result, nil
}
