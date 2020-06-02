package config

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	log "github.com/cihub/seelog"
	cfgsvc "github.com/HailoOSS/service/config"
)

// Bootstrap loads minimal viable config needed for config service
// specifically c* hosts from H2_CONFIG_SERVICE_CASSANDRA
// and authentication settings from:
//   H2_CONFIG_SERVICE_CASSANDRA_AUTH_ENABLED
//   H2_CONFIG_SERVICE_CASSANDRA_AUTH_USERNAME
//   H2_CONFIG_SERVICE_CASSANDRA_AUTH_PASSWORD
func Bootstrap() {

	// Hosts in a comma separated string
	hosts := strings.Split(os.Getenv("H2_CONFIG_SERVICE_CASSANDRA"), ",")

	// Cassandra Authentication
	authenabled := os.Getenv("H2_CONFIG_SERVICE_CASSANDRA_AUTH_ENABLED")
	authuser := os.Getenv("H2_CONFIG_SERVICE_CASSANDRA_AUTH_USERNAME")
	authpass := os.Getenv("H2_CONFIG_SERVICE_CASSANDRA_AUTH_PASSWORD")

	// Build config
	bootstrapCfg := map[string]interface{}{
		"hailo": map[string]interface{}{
			"service": map[string]interface{}{
				"cassandra": map[string]interface{}{
					"hosts": hosts,
					"authentication": map[string]interface{}{
						"enabled": authenabled,
						"keyspaces": map[string]interface{}{
							"configservice": map[string]interface{}{
								"username": authuser,
								"password": authpass,
							},
						},
					},
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
