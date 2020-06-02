package multiclient

import (
	"sync"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
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
