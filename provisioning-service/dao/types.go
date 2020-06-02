package dao

type ServiceType uint

const (
	ServiceTypeProcess   ServiceType = 0
	ServiceTypeContainer ServiceType = 1
)

var ServiceTypeByName = map[ServiceType]string{
	0: "Process",
	1: "Container",
}

type ProvisionedService struct {
	ServiceName     string
	ServiceVersion  uint64
	MachineClass    string
	NoFileSoftLimit uint64
	NoFileHardLimit uint64
	ServiceType     ServiceType
}

type ProvisionedServices []*ProvisionedService

func (ps *ProvisionedService) matches(name string, version uint64, typ ServiceType) bool {
	if ps.ServiceName == name && ps.ServiceVersion == version && ps.ServiceType == typ {
		return true
	}

	return false
}

// Contains will check if a list of provisioned services contains a specified
// service (specified by name, version and type)
func (ps ProvisionedServices) Contains(name string, version uint64, typ ServiceType) bool {
	for _, service := range ps {
		if service.matches(name, version, typ) {
			return true
		}
	}

	return false
}
