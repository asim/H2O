// +build integration

package s3

import (
	"os"
	"testing"

	"github.com/hailo-platform/H2O/provisioning-service/dao"
)

func TestDownload(t *testing.T) {
	s := &dao.ProvisionedService{ServiceName: "com.hailo-platform/H2O.service.banning", ServiceVersion: 20130719163756, MachineClass: "A"}

	filename, err := Download(s)
	if err != nil {
		t.Error("Download error: ", err)
	}

	// make sure it exists
	if _, err := os.Stat(filename); err != nil {
		t.Error(err)
	}

	if err := os.Remove(filename); err != nil {
		t.Error(err)
	}
}
