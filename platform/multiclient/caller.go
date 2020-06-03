package multiclient

import (
	"sync"

	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/errors"
)

type Caller func(req *client.Request, rsp proto.Message) errors.Error

var (
	caller = PlatformCaller()
	mtx    sync.RWMutex
)

// SetCaller is provided for tests to use to swap out the default calling mechanism
// for an alternative (eg: stubbed caller)
func SetCaller(c Caller) {
	mtx.Lock()
	defer mtx.Unlock()
	caller = c
}

func getCaller() Caller {
	mtx.RLock()
	defer mtx.RUnlock()
	return caller
}
