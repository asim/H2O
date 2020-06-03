package statusmonitor

import (
	zk "github.com/hailo-platform/H2O/service/zookeeper"
	gozk "github.com/hailo-platform/H2O/go-zookeeper/zk"
)

const (
	lockPath = "/hailo-2-api-az-failover"
)

// lock creates an ephemeral lock
func lock(az string) (string, error) {
	bdata := []byte(az)
	l, err := zk.Create(lockPath, bdata, gozk.FlagEphemeral, gozk.WorldACL(gozk.PermAll))
	return l, err
}

// unlock removes existing ephemeral lock
func unlock(lockPath string) error {
	err := zk.Delete(lockPath, 0)
	return err
}

// readLock reads and returns data from a zk lock file
func readLock() (string, error) {
	b, _, err := zk.Get(lockPath)
	return string(b), err
}
