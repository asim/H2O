// +build integration

package process

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

func createTestFile(filename string) {
	// make sure folder exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.ModeDir|os.ModePerm); err != nil {
		log.Fatalln("Error: ", err)
	}

	const contents = "#!/bin/bash\n" + "sleep 1s"
	err := ioutil.WriteFile(filename, []byte(contents), 0774)
	if err != nil {
		log.Fatalln("Error: ", err)
	}
}

func TestStartStop(t *testing.T) {
	filename := path.Join(exeDir, "com.hailo-platform/H2O.service.provisioning.testprocess-20130102030405")
	createTestFile(filename)
	defer os.Remove(filename)

	if err := Start("com.hailo-platform/H2O.service.provisioning.testprocess", 20130102030405, 1024, 4096); err != nil {
		t.Error("Error testing Start():", err)
	}

	numInstances, err := CountRunningInstances("com.hailo-platform/H2O.service.provisioning.testprocess", 20130102030405)
	if err != nil {
		t.Error("Error:", err)
	}
	if numInstances != 1 {
		t.Error("Expecting our made up provisioned service to be running")
	}

	if err := Stop("com.hailo-platform/H2O.service.provisioning.testprocess", 20130102030405); err != nil {
		t.Error("Error testing Stop():", err)
	}

	time.Sleep(time.Second * 2) // it takes some time for the service to actually stop

	numInstances, err = CountRunningInstances("com.hailo-platform/H2O.service.provisioning.testprocess", 20130102030405)
	if err != nil {
		t.Error("Error:", err)
	}
	if numInstances != 0 {
		t.Error("Not expecting our made up provisioned service to be running")
	}
}
