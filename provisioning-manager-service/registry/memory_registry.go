package registry

import (
	"fmt"
	"github.com/HailoOSS/provisioning-manager-service/domain"
	"sync"
)

type MemoryRegistry struct {
	sync.RWMutex
	provisioners map[string]*domain.Provisioner
}

func NewMemoryRegistry() Registry {
	return &MemoryRegistry{
		provisioners: make(map[string]*domain.Provisioner),
	}
}

func (m *MemoryRegistry) Get(p *domain.Provisioner) (*domain.Provisioner, error) {
	m.RLock()
	defer m.RUnlock()
	if pp, ok := m.provisioners[p.Hostname]; ok {
		return pp, nil
	}
	return nil, fmt.Errorf("Not found")
}

func (m *MemoryRegistry) Insert(p *domain.Provisioner) error {
	m.Lock()
	defer m.Unlock()
	m.provisioners[p.Hostname] = p
	return nil
}

func (m *MemoryRegistry) Delete(p *domain.Provisioner) error {
	m.Lock()
	defer m.Unlock()
	delete(m.provisioners, p.Hostname)
	return nil
}

func (m *MemoryRegistry) List() ([]*domain.Provisioner, error) {
	m.RLock()
	defer m.RUnlock()

	var provisioners []*domain.Provisioner

	for _, p := range m.provisioners {
		provisioners = append(provisioners, p)
	}

	return provisioners, nil
}

func (m *MemoryRegistry) Filtered(fn Filter) ([]*domain.Provisioner, error) {
	m.RLock()
	defer m.RUnlock()

	var provisioners []*domain.Provisioner

	for _, p := range m.provisioners {
		if !fn(p) {
			continue
		}
		provisioners = append(provisioners, p)
	}

	return provisioners, nil
}
