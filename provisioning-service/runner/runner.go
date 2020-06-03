package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/provisioning-service/dao"
)

const (
	checkInterval = 5
)

var (
	myClass string
	docker  bool
)

func init() {
	myClass = os.Getenv("H2O_MACHINE_CLASS")
	if len(myClass) == 0 {
		myClass = "default"
	}
}

func Run() {
	go run()
}

// run the loop - this continually checks the list of what should be running,
// what is running, and brings those two into alignment by starting and
// stopping services
func run() {
	log.Info("Provisioning service running on class '", myClass, "'")

	docker = isDockerized()

	if docker {
		j := &janitor{
			maxStoppedTime: 60 * time.Minute,
			sleepInterval:  30 * time.Second,
		}
		go j.clean()
	}

	ticker := time.NewTicker(checkInterval * time.Second)
	for {
		select {
		case <-ticker.C:
			check()
		}
	}
}

func check() {
	log.Debug("Checking running services ...")

	services, err := dao.Services(myClass)
	if err != nil {
		log.Warn("Error fetching provisioned services list: ", err)
		return
	}

	log.Debugf("Found %d services that should be running", len(services))

	// Loop through our processes
	if err := startMissingProcesses(services); err != nil {
		log.Warnf("Error starting missing services: %v ", err)
		return
	}

	if err := stopExtraProcesses(services); err != nil {
		log.Warnf("Error stopping extra services: %v ", err)
		return
	}

	if docker {
		// Loop through our containers
		// FIXME: Should split in parallel runners
		if err := startMissingContainers(services); err != nil {
			log.Warnf("Error starting missing containers: %v", err)
			return
		}

		if err := stopExtraContainers(services); err != nil {
			log.Warnf("Error stopping extra services: %v", err)
			return
		}
	}
}

func splitLast(input string, char string) (string, string, error) {
	last := strings.LastIndex(input, char)
	if last == -1 {
		return input, "", fmt.Errorf("Splitting \"%s\" by \"%s\" did not result in at least 2 parts", input, char)
	}
	return input[:last], input[last+1:], nil
}

func splitProcessName(processName string) (serviceName string, serviceVersion uint64, err error) {
	_, filenameOnly := path.Split(processName)
	serviceName, stringVersion, err := splitLast(filenameOnly, "-")
	if err != nil {
		return "", 0, err
	}

	serviceVersion, err = strconv.ParseUint(stringVersion, 10, 64)
	if err != nil {
		return "", 0, err
	}

	return serviceName, serviceVersion, nil
}

func isDockerized() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	return true
}
