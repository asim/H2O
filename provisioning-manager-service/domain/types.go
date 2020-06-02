package domain

import (
	"github.com/HailoOSS/protobuf/proto"
	pproto "github.com/HailoOSS/provisioning-manager-service/proto"
	"time"
)

type Service struct {
	Name       string
	Version    string
	Usage      *Resource
	Allocation *Resource
}

type Resource struct {
	Cpu    float64
	Memory uint64
	Disk   uint64
}

type Machine struct {
	Cores  uint64
	Memory uint64
	Disk   uint64
	Usage  *Resource
}

type Provisioner struct {
	Id           string
	Version      string
	Hostname     string
	IpAddress    string
	AzName       string
	MachineClass string
	Started      time.Time
	LastUpdate   time.Time
	Machine      *Machine
	Processes    []*Service
	Containers   []*Service
}

func (p *Provisioner) ToProto(verbose bool) *pproto.Provisioner {
	pp := &pproto.Provisioner{
		Id:           proto.String(p.Id),
		Version:      proto.String(p.Version),
		Hostname:     proto.String(p.Hostname),
		IpAddress:    proto.String(p.IpAddress),
		AzName:       proto.String(p.AzName),
		MachineClass: proto.String(p.MachineClass),
		Machine: &pproto.Machine{
			Cores:  proto.Uint64(p.Machine.Cores),
			Memory: proto.Uint64(p.Machine.Memory),
			Disk:   proto.Uint64(p.Machine.Disk),
		},
		Started:    proto.Uint64(uint64(p.Started.Unix())),
		LastUpdate: proto.Uint64(uint64(p.LastUpdate.Unix())),
	}

	if verbose {
		pp.Machine.Usage = &pproto.Resource{
			Cpu:    proto.Float64(p.Machine.Usage.Cpu),
			Memory: proto.Uint64(p.Machine.Usage.Memory),
			Disk:   proto.Uint64(p.Machine.Usage.Disk),
		}
		for _, proc := range p.Processes {
			pp.Processes = append(pp.Processes, &pproto.Service{
				Name:    proto.String(proc.Name),
				Version: proto.String(proc.Version),
				Usage: &pproto.Resource{
					Cpu:    proto.Float64(proc.Usage.Cpu),
					Memory: proto.Uint64(proc.Usage.Memory),
				},
			})
		}
		for _, cont := range p.Containers {
			pp.Containers = append(pp.Containers, &pproto.Service{
				Name:    proto.String(cont.Name),
				Version: proto.String(cont.Version),
				Usage: &pproto.Resource{
					Cpu:    proto.Float64(cont.Usage.Cpu),
					Memory: proto.Uint64(cont.Usage.Memory),
				},
				Allocation: &pproto.Resource{
					Cpu:    proto.Float64(cont.Allocation.Cpu),
					Memory: proto.Uint64(cont.Allocation.Memory),
				},
			})
		}
	}

	return pp
}
