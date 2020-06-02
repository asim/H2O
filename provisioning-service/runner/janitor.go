package runner

import (
	"time"

	log "github.com/cihub/seelog"

	"github.com/HailoOSS/provisioning-service/container"
)

type janitor struct {
	maxStoppedTime time.Duration
	sleepInterval  time.Duration
}

// clean wakes up every 5 minutes and checks the state of the idle containers
func (j *janitor) clean() {
	log.Infof("Starting the container janitor loop")

	for {

		time.Sleep(j.sleepInterval)
		log.Debugf("Checking for stale containers")

		now := time.Now().UTC()
		containers, err := container.ListContainers(true)
		if err != nil {
			log.Errorf("Unable to list containers %v", err)
			continue
		}

		for _, c := range containers {
			s, err := container.InspectContainer(c.ID)
			if err != nil {
				log.Warnf("Can't lookup container %s:%v", c.ID, err)
				continue
			}
			if !s.State.Running {
				delta := now.Sub(s.State.FinishedAt.UTC())
				if delta.Seconds() > j.maxStoppedTime.Seconds() {
					err := container.RemoveContainer(s.ID)
					if err != nil {
						log.Errorf("Unable to remove container %s: %v", s.Name, err)
						continue
					}
					log.Infof("Removed container %s", s.Name)

					err = container.RemoveImage(s.Image)
					if err != nil {
						log.Errorf("Unable to remove image %s: %v", s.Image, err)
						continue
					}
					log.Infof("Removed image %s", s.Image)
				}
			}
		}

	}

}
