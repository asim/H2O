package runlevels

import (
	"sync"
	"time"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/platform/util"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-manager-service/dao"
	levelsproto "github.com/HailoOSS/provisioning-manager-service/proto"
	descproto "github.com/HailoOSS/provisioning-manager-service/proto/describerunlevels"
	pproto "github.com/HailoOSS/provisioning-manager-service/proto/provisioned"
)

const (
	DefaultRunLevel = 5
	HaltRunLevel    = 0
	MinRunLevel     = 0
	MaxRunLevel     = 5
)

var levels map[int]runLevel = map[int]runLevel{
	0: runLevel{0, "Halt", "No services running."},
	1: runLevel{1, "Platform", "Platform is running."},
	2: runLevel{2, "Critical", "Critical path services running."},
	3: runLevel{3, "Essential", "Essential services running. Including async queue processing."},
	4: runLevel{4, "Degraded", "Degraded mode. Removes non essential internal tools."},
	5: runLevel{5, "All", "All services running"},
}

var (
	defaultManager = newRunLevelManager()
)

type runLevel struct {
	Level       int64
	Name        string
	Description string
}

type runLevelManager struct {
	sync.RWMutex
	runLevel int64
	services map[string][6]bool
}

func newRunLevelManager() *runLevelManager {
	return &runLevelManager{
		runLevel: DefaultRunLevel,
		services: make(map[string][6]bool),
	}
}

func (r *runLevelManager) filter(services []*pproto.Service) []*pproto.Service {
	// Get the current run level and service run levels
	r.RLock()
	serviceRunLevels := r.services
	runLevel := r.runLevel
	r.RUnlock()

	// Filter list to what should run in the run level
	var runningServices []*pproto.Service

	for _, ps := range services {
		// If the provisioning manager is in the list then we ALWAYS add it.
		if ps.GetServiceName() == server.Name {
			runningServices = append(runningServices, ps)
			continue
		}

		// Now check everything else.
		service, ok := serviceRunLevels[ps.GetServiceName()]
		// If there is no runlevel for this service then run anyway, this is to
		// prevent new services which have not yet had a run level configured
		// from being deprovisioned. Also run the service if the service has
		// the correct runlevel
		if !ok || service[runLevel] {
			runningServices = append(runningServices, ps)
		}
	}

	return runningServices
}

func (r *runLevelManager) getRunLevel() int64 {
	r.RLock()
	defer r.RUnlock()
	return r.runLevel
}

func (r *runLevelManager) getServices() map[string][6]bool {
	r.RLock()
	defer r.RUnlock()
	return r.services
}

func (r *runLevelManager) updateRunLevel() {
	region := util.GetAwsRegionName()
	runLevel, err := dao.ReadRunLevel(region)
	if err != nil {
		log.Errorf("Failed to read run level from cassandra: %v", err)
		return
	}

	r.Lock()
	defer r.Unlock()
	r.runLevel = runLevel.Level
}

func (r *runLevelManager) updateServices() {
	runLevels, err := dao.ReadServiceRunLevels()
	if err != nil {
		log.Errorf("Failed to read service run levels from cassandra: %v", err)
		return
	}

	services := make(map[string][6]bool)

	for _, service := range runLevels {
		services[service.ServiceName] = service.Levels
	}

	r.Lock()
	defer r.Unlock()
	r.services = services
}

func DescribeProto() *descproto.Response {
	var runLevels []*descproto.RunLevel

	for _, level := range levels {
		lvl := levelsproto.Level(level.Level)
		runLevels = append(runLevels, &descproto.RunLevel{
			Level:       &lvl,
			Description: proto.String(level.Description),
		})
	}

	return &descproto.Response{
		RunLevels: runLevels,
	}
}

func Filter(services []*pproto.Service) []*pproto.Service {
	return defaultManager.filter(services)
}

func RunLevel() int64 {
	return defaultManager.getRunLevel()
}

func Services() map[string][6]bool {
	return defaultManager.getServices()
}

func Run() {
	defaultManager.updateRunLevel()
	defaultManager.updateServices()
	tick := time.NewTicker(time.Minute)

	for {
		select {
		case <-tick.C:
			defaultManager.updateRunLevel()
			defaultManager.updateServices()
		}
	}
}

func Setup() {
	go Run()
}
