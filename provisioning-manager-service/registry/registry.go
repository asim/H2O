package registry

import (
	"github.com/HailoOSS/provisioning-manager-service/domain"
)

type Registry interface {
	Get(*domain.Provisioner) (*domain.Provisioner, error)
	Insert(*domain.Provisioner) error
	Delete(*domain.Provisioner) error
	List() ([]*domain.Provisioner, error)
	Filtered(Filter) ([]*domain.Provisioner, error)
}

var (
	DefaultRegistry = NewMemoryRegistry()
)

func Get(p *domain.Provisioner) (*domain.Provisioner, error) {
	return DefaultRegistry.Get(p)
}

func Insert(p *domain.Provisioner) error {
	return DefaultRegistry.Insert(p)
}

func Delete(p *domain.Provisioner) error {
	return DefaultRegistry.Delete(p)
}

func List() ([]*domain.Provisioner, error) {
	return DefaultRegistry.List()
}

func Filtered(fn Filter) ([]*domain.Provisioner, error) {
	return DefaultRegistry.Filtered(fn)
}
