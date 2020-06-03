// +build integration

package runner

import (
	"github.com/hailo-platform/H2O/provisioning-service/dao"
	proc "github.com/hailo-platform/H2O/provisioning-service/process"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func createTestFile(filename string) {
	const contents = "#!/bin/bash\n" + "sleep 1s"
	err := ioutil.WriteFile(filename, []byte(contents), 0774)
	if err != nil {
		log.Fatalln("Error: ", err)
	}
}

func TestSplitProcessName(t *testing.T) {
	const filename = "/opt/hailo/bin/com.hailo-platform/H2O.service.provisioning.testsplitprocessname-20130102030405"
	serviceName, seviceVersion, err := splitProcessName(filename)
	if err != nil {
		log.Fatalln("Error: ", err)
	}

	if serviceName != "com.hailo-platform/H2O.service.provisioning.testsplitprocessname" {
		t.Error("Error testing splitProcessName() - serviceName was wrong")
	}

	if seviceVersion != 20130102030405 {
		t.Error("Error testing splitProcessName() - seviceVersion was wrong")
	}
}

func TestStopExtraProcesses(t *testing.T) {
	const filename = "/opt/hailo/bin/com.hailo-platform/H2O.service.provisioning.teststopextra-20130102030405"
	createTestFile(filename)
	defer os.Remove(filename)

	extraP := &dao.ProvisionedService{ServiceName: "com.hailo-platform/H2O.service.provisioning.teststopextra", ServiceVersion: 20130102030405, MachineClass: "A"}

	if err := proc.Start("com.hailo-platform/H2O.service.provisioning.teststopextra", 20130102030405, 1024, 4096); err != nil {
		t.Error("Error starting service:", err)
	}

	numInstances, err := proc.CountRunningInstances("com.hailo-platform/H2O.service.provisioning.teststopextra", 20130102030405)
	if err != nil {
		t.Error("Error:", err)
	}
	if numInstances != 1 {
		t.Error("Expecting our made up provisioned service to be running")
	}

	pss := make(dao.ProvisionedServices, 0)
	if err := stopExtraProcesses(pss); err != nil {
		t.Error("Error testing StopExtraProcesses(): ", err)
	}

	numInstances, err = proc.CountRunningInstances(extraP.ServiceName, extraP.ServiceVersion)
	if err != nil {
		log.Fatalln("Error: ", err)
	}

	if numInstances != 0 {
		t.Error("Error testing StopExtraProcesses() - extra process did not stop")
	}
}
