package runner

import (
	"fmt"
	"strconv"

	log "github.com/cihub/seelog"

	"github.com/hailo-platform/H2O/go-hailo-lib/multierror"
	"github.com/hailo-platform/H2O/provisioning-service/container"
	"github.com/hailo-platform/H2O/provisioning-service/dao"
	"github.com/hailo-platform/H2O/provisioning-service/event"
)

func startMissingContainers(provisionedServices dao.ProvisionedServices) error {
	me := multierror.New()

	for _, service := range provisionedServices {
		if service.ServiceType != dao.ServiceTypeContainer {
			continue
		}

		name := combineNameVersion(service.ServiceName, service.ServiceVersion)
		version := strconv.Itoa(int(service.ServiceVersion))

		if container.IsRunning(name) {
			continue
		}

		// Load the service dependencies
		// We need to expose extra functionality for that
		// if err := deps.Load(service.ServiceName); err != nil {
		// 	log.Criticalf("Failed to load dependencies for service %s: %v", service.ServiceName, err)
		// }
		log.Debugf("Container %s:%s is not yet running", service.ServiceName, version)
		if !container.IsDownloaded(service.ServiceName, version) {
			log.Debugf("Container %s:%s is not yet downloaded", service.ServiceName, version)
			err := container.Download(service.ServiceName, version)
			if err != nil {
				// log error and continue, so we don't block other provisioned services
				msg := fmt.Sprintf("Container image could not be downloaded: %v", err)
				log.Warnf(msg)
				event.ProvisionError(service.ServiceName, service.ServiceVersion, msg)
				me.Add(err)
				continue
			}
			log.Debugf("Downloaded image: %s:%s!", service.ServiceName, version)
		}

		if err := container.Start(service.ServiceName, version, nil); err != nil {
			msg := fmt.Sprintf("Container could not be started: %v", err)
			log.Warnf(msg)
			event.ProvisionError(service.ServiceName, service.ServiceVersion, msg)
			me.Add(err)
			continue
		}

		log.Debugf("Started container %s:%s!", service.ServiceName, version)
		event.Provisioned(service.ServiceName, service.ServiceVersion)
	}

	if me.AnyErrors() {
		return me
	}

	return nil
}

func stopExtraContainers(provisionedServices dao.ProvisionedServices) error {
	// stop any services that are running but shouldn't be
	runningContainerNames, err := container.ListRunning("com.hailo-platform/H2O")
	if err != nil {
		return err
	}

	me := multierror.New()

	for _, runningContainerName := range runningContainerNames {
		runningName, runningVersion, err := splitProcessName(runningContainerName)
		if err != nil {
			me.Add(err)
			continue
		}

		if provisionedServices.Contains(runningName, runningVersion, dao.ServiceTypeContainer) {
			continue
		}

		if err := container.Stop(runningContainerName, 0); err != nil {
			msg := fmt.Sprintf("Container %s could not be stopped: %v", runningContainerName, err)
			log.Warnf(msg)
			event.DeprovisionError(runningName, runningVersion, msg)
			me.Add(err)
			continue
		}

		log.Debugf("Stopped container %s", runningContainerName)
		event.Deprovisioned(runningName, runningVersion)
	}

	if me.AnyErrors() {
		return me
	}

	return nil
}

func combineNameVersion(serviceName string, serviceVersion uint64) string {
	return serviceName + "-" + strconv.Itoa(int(serviceVersion))
}
