package auther

import (
	"fmt"

	"github.com/hailo-platform/H2O/service/sync"
)

const (
	lockPath = "%s/%s-%s"
)

func lockDeviceUser(authMech, deviceType, userId string) (sync.Lock, error) {
	return sync.RegionLock([]byte(fmt.Sprintf(lockPath, authMech, deviceType, userId)))
}
