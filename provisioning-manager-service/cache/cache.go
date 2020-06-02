package cache

import (
	"fmt"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/provisioning-manager-service/dao"
	pproto "github.com/HailoOSS/provisioning-manager-service/proto/provisioned"
)

const (
	updateInterval = time.Second * 5
)

type cacheManager struct {
	sync.RWMutex
	services []*pproto.Service
}

var (
	defaultManager = newCacheManager()
)

func newCacheManager() *cacheManager {
	return &cacheManager{}
}

func (c *cacheManager) provisioned(serviceName, machineClass string) ([]*pproto.Service, error) {
	c.RLock()
	defer c.RUnlock()

	// safety, never return 0
	if len(c.services) == 0 {
		return nil, fmt.Errorf("Cache is empty")
	}

	// return all
	if len(serviceName) == 0 && len(machineClass) == 0 {
		return c.services, nil
	}

	var provisioned []*pproto.Service
	for _, service := range c.services {
		if len(serviceName) > 0 && service.GetServiceName() != serviceName {
			continue
		}
		if len(machineClass) > 0 && service.GetMachineClass() != machineClass {
			continue
		}
		provisioned = append(provisioned, service)
	}
	return provisioned, nil
}

func (c *cacheManager) update() error {
	services, err := dao.Provisioned("")
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	var provisioned []*pproto.Service
	for _, service := range services {
		provisioned = append(provisioned, service.ToProto())
	}
	c.services = provisioned
	return nil
}

func (c *cacheManager) run() {
	ticker := time.NewTicker(updateInterval)

	for {
		select {
		case <-ticker.C:
			if err := c.update(); err != nil {
				log.Errorf("Error retrieving provisioned services: %v", err)
			}
		}
	}
}

func Provisioned(serviceName, machineClass string) ([]*pproto.Service, error) {
	return defaultManager.provisioned(serviceName, machineClass)
}

func Run() {
	go defaultManager.run()
}
