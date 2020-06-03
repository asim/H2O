package runner

import (
	"fmt"

	log "github.com/cihub/seelog"

	"github.com/hailo-platform/H2O/go-hailo-lib/multierror"
	"github.com/hailo-platform/H2O/provisioning-service/dao"
	"github.com/hailo-platform/H2O/provisioning-service/deps"
	"github.com/hailo-platform/H2O/provisioning-service/event"
	"github.com/hailo-platform/H2O/provisioning-service/pkgmgr"
	"github.com/hailo-platform/H2O/provisioning-service/process"
)

func startMissingProcesses(provisionedServices dao.ProvisionedServices) error {
	me := multierror.New()

	// start up any services that aren't running but should be
	runningProcesses, err := process.ListRunning("com.hailocab")
	if err != nil {
		return err
	}

	for _, service := range provisionedServices {
		if service.ServiceType != dao.ServiceTypeProcess {
			continue
		}

		numInstances := process.CachedCountRunningInstances(service.ServiceName, service.ServiceVersion, runningProcesses)

		if numInstances > 0 {
			continue
		}

		// Load the service dependencies
		if err := deps.Load(service.ServiceName); err != nil {
			log.Criticalf("Failed to load dependencies for service %s: %v", service.ServiceName, err)
		}

		log.Debugf("Service %v is not yet running", service)
		if dl, _ := pkgmgr.IsDownloaded(service); !dl {
			log.Debugf("Service %v is not yet downloaded", service)
			_, err := pkgmgr.Download(service)
			if err != nil {
				// log error and continue, so we don't block other provisioned services
				msg := fmt.Sprintf("Provisioned service could not be downloaded: %v", err)
				log.Warnf(msg)
				event.ProvisionError(service.ServiceName, service.ServiceVersion, msg)
				me.Add(err)
				// Delete downloaded file if it exists
				if err := pkgmgr.Delete(service); err != nil {
					log.Warnf("Failed deleting file after failing to download: %v", err)
				}
				continue
			}
			log.Debugf("Downloaded service: %v!", service)
		}

		// Verify the binary, if it fails, delete and wait for the next cycle
		if err := pkgmgr.VerifyBinary(service); err != nil {
			msg := fmt.Sprintf("Failed to verify binary, will be deleted, err: %v", err)
			log.Criticalf(msg)
			event.ProvisionError(service.ServiceName, service.ServiceVersion, msg)
			if err := pkgmgr.Delete(service); err != nil {
				log.Warnf("Failed to delete binary: %v", err)
			}
			continue
		}

		if err := process.Start(service.ServiceName, service.ServiceVersion, service.NoFileSoftLimit, service.NoFileHardLimit); err != nil {
			msg := fmt.Sprintf("Provisioned service could not be started: %v", err)
			log.Warnf(msg)
			event.ProvisionError(service.ServiceName, service.ServiceVersion, msg)
			me.Add(err)
			continue
		}

		log.Debugf("Started service %v!", service)
		event.Provisioned(service.ServiceName, service.ServiceVersion)
	}

	if me.AnyErrors() {
		return me
	}

	return nil
}

func stopExtraProcesses(provisionedServices dao.ProvisionedServices) error {
	// stop any services that are running but shouldn't be
	runningProcessNames, err := process.ListRunning("com.hailocab")
	if err != nil {
		return err
	}

	me := multierror.New()

	for _, runningProcessName := range runningProcessNames {
		runningName, runningVersion, err := splitProcessName(runningProcessName)
		if err != nil {
			me.Add(err)
			continue
		}

		if provisionedServices.Contains(runningName, runningVersion, dao.ServiceTypeProcess) {
			continue
		}

		if err := process.Stop(runningName, runningVersion); err != nil {
			event.DeprovisionError(runningName, runningVersion, err.Error())
			me.Add(err)
			continue
		}

		if err := pkgmgr.Delete(&dao.ProvisionedService{
			ServiceName:    runningName,
			ServiceVersion: runningVersion,
		}); err != nil {
			me.Add(err)
		}

		event.Deprovisioned(runningName, runningVersion)
	}

	if me.AnyErrors() {
		return me
	}

	return nil
}
