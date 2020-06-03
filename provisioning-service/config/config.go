package config

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	log "github.com/cihub/seelog"
	cfgsvc "github.com/hailo-platform/H2O/service/config"
)

// Bootstrap loads minimal viable config needed for the provisioning service - specifically c* hosts from H2_CONFIG_SERVICE_CASSANDRA
func Bootstrap() {
	hosts := strings.Split(os.Getenv("H2_CONFIG_SERVICE_CASSANDRA"), ",")
	bootstrapCfg := map[string]interface{}{
		"hailo": map[string]interface{}{
			"service": map[string]interface{}{
				"cassandra": map[string]interface{}{
					"hosts": hosts,
				},
			},
		},
	}
	b, _ := json.Marshal(bootstrapCfg)
	rdr := bytes.NewReader(b)
	if err := cfgsvc.Load(rdr); err != nil {
		log.Criticalf("Failed to bootstrap config service C* config: %v", err)
	} else {
		log.Infof("Bootstrapped C* config: %v", bootstrapCfg)
	}
}
