// +build integration

package s3

import (
	"os"
	"testing"

	"github.com/HailoOSS/provisioning-service/dao"
)

func TestDownload(t *testing.T) {
	s := &dao.ProvisionedService{ServiceName: "com.HailoOSS.service.banning", ServiceVersion: 20130719163756, MachineClass: "A"}

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
