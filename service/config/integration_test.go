// +build integration

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadFromConfigService(t *testing.T) {
	setupTest()
	LoadFromService("testservice")
	s := AtPath("configService", "hash").AsString("default")
	assert.NotEqual(t, "default", s, "Failed to load config from config service")
}
