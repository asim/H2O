package scheduler

import (
	"time"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/provisioning-manager-service/registry"
)

const (
	interval = time.Minute
	reapAge  = time.Minute * 2
)

type scheduler struct{}

var (
	defaultScheduler = newScheduler()
)

func newScheduler() *scheduler {
	return &scheduler{}
}

func (s *scheduler) reap() {
	provisioners, err := registry.List()
	if err != nil {
		log.Errorf("[scheduler] Failed to get registry list: %v", err)
		return
	}

	for _, provisioner := range provisioners {
		if lu := time.Since(provisioner.LastUpdate); lu.Seconds() > reapAge.Seconds() {
			log.Info("[scheduler] Reaping %s from registry. Last update %d seconds", provisioner.Id, lu.Seconds())
			registry.Delete(provisioner)
		}
	}
}

func (s *scheduler) run() {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			s.reap()
		}
	}
}

func Run() {
	go defaultScheduler.run()
}
