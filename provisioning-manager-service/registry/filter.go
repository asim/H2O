package registry

import (
	"github.com/hailo-platform/H2O/provisioning-manager-service/domain"
)

type Filter func(p *domain.Provisioner) bool

func FilterMachineClass(mc string) Filter {
	return func(p *domain.Provisioner) bool {
		if p.MachineClass == mc {
			return true
		}
		return false
	}
}
