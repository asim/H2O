package dao

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	pproto "github.com/HailoOSS/provisioning-manager-service/proto/provisioned"
)

const (
	cacheFile = "/opt/hailo/var/cache/provisioned.json"
)

type loader struct {
	mtx         sync.RWMutex
	initialised bool
	hash        string
	services    ProvisionedServices
}

var (
	defaultLoader = newLoader()
)

func init() {
	if services, err := defaultLoader.load(); err == nil {
		defaultLoader.cache(services)
	}
}

func getProvisionedServices(machineClass string) (ProvisionedServices, error) {
	request, err := server.ScopedRequest("com.HailoOSS.kernel.provisioning-manager", "provisioned", &pproto.Request{
		MachineClass: proto.String(machineClass),
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to create provisioning manager provisioned request: %v", err)
	}

	response := &pproto.Response{}
	if err := client.Req(request, response); err != nil {
		return nil, fmt.Errorf("Provisioning manager provisioned request failed: %v", err)
	}

	var provisioned ProvisionedServices
	for _, service := range response.GetServices() {
		provisioned = append(provisioned, &ProvisionedService{
			ServiceName:     service.GetServiceName(),
			ServiceVersion:  service.GetServiceVersion(),
			MachineClass:    service.GetMachineClass(),
			NoFileSoftLimit: service.GetNoFileSoftLimit(),
			NoFileHardLimit: service.GetNoFileHardLimit(),
			ServiceType:     ServiceType(service.GetServiceType()),
		})
	}
	return provisioned, nil
}

func newLoader() *loader {
	return &loader{}
}

func (l *loader) load() (ProvisionedServices, error) {
	b, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var services ProvisionedServices
	err = json.Unmarshal(b, &services)
	if err != nil {
		return nil, err
	}

	return services, nil
}

func (l *loader) save() error {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	b, err := json.Marshal(l.services)
	if err != nil {
		return err
	}

	dir := filepath.Dir(cacheFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	if err := ioutil.WriteFile(cacheFile, b, 0644); err != nil {
		return err
	}

	return nil
}

func (l *loader) cache(services ProvisionedServices) {
	l.mtx.Lock()
	l.services = services
	l.hash = fmt.Sprintf("%x", services)
	l.initialised = true
	l.mtx.Unlock()
}

func (l *loader) hasChanged(services ProvisionedServices) bool {
	l.mtx.RLock()
	defer l.mtx.RUnlock()

	if hash := fmt.Sprintf("%x", services); l.hash != hash {
		return true
	}

	return false
}

func (l *loader) getCachedServices(machineClass string) (ProvisionedServices, error) {
	l.mtx.RLock()
	defer l.mtx.RUnlock()

	if l.initialised {
		return l.services, nil
	}

	return nil, fmt.Errorf("No loaded services")
}

func (l *loader) getServices(machineClass string) (ProvisionedServices, error) {
	// load from provisioning manager
	services, err := getProvisionedServices(machineClass)
	if err == nil {
		if l.hasChanged(services) {
			l.cache(services)
			if err := l.save(); err != nil {
				log.Warnf("Error saving provisioned services to disk: %v", err)
			}
		}
		return services, nil
	} else {
		log.Errorf("Unable to get services list from prov manager: %v", err)
	}

	// load from cache
	services, err = l.getCachedServices(machineClass)
	if err == nil {
		return services, nil
	}

	// load from disk
	services, err = l.load()
	if err == nil {
		l.cache(services)
		return services, nil
	}

	return nil, fmt.Errorf("No loaded services: %v", err)
}

func CachedServices(machineClass string) (ProvisionedServices, error) {
	return defaultLoader.getCachedServices(machineClass)
}

func Services(machineClass string) (ProvisionedServices, error) {
	return defaultLoader.getServices(machineClass)
}
