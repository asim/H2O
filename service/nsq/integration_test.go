// +build integration

// (relies on having running NSQ and access to config service correctly configured)

package nsq

import (
	"fmt"
	"github.com/HailoOSS/service/config"
	"testing"
)

func TestPub(t *testing.T) {
	config.LoadFromService("testservice")
	s := config.AtPath("configService", "hash").AsString("default")
	if s == "default" {
		t.Fatal("Failed to load config from config service")
	}
	err := Publish("testtopic", []byte("This is my payload"))
	if err != nil {
		t.Error(fmt.Sprintf("Failed to PUB: %v", err))
	}
}
