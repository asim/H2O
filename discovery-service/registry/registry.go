package registry

const (
	rootNode     = "/discovery-service"
	instanceNode = "/discovery-service/%v"
)

var (
	local  *localReg
	region *regionReg
)

func Init() {
	local = newLocalReg()
	region = newRegionReg()
}

// Register registers an instance with this discovery service
// Upon successful registration, this instance of the discovery service will
// send periodic heartbeats to check the service it alive, and is then responsible
// for removing it if it dies
func Register(instance *Instance) error {
	return local.add(instance)
}

// Unregister removes an instance, by ID, plus any endpoints running within this instance
func Unregister(instanceId string) error {
	return local.remove(instanceId)
}

func Hosts() ([]string, error) {
	return []string{}, nil
}

// AllInstances returns a snapshot of all instances for further in-memory manipulation
func AllInstances() Instances {
	return region.allInstances()
}
