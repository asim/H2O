package handler

import (
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	"github.com/HailoOSS/provisioning-manager-service/domain"
	"github.com/HailoOSS/provisioning-manager-service/registry"
	iproto "github.com/HailoOSS/provisioning-service/proto"
	"time"
)

func protoToProvisioner(p *iproto.Info) *domain.Provisioner {
	var processes []*domain.Service
	var containers []*domain.Service

	for _, proc := range p.GetProcesses() {
		processes = append(processes, &domain.Service{
			Name:    proc.GetName(),
			Version: proc.GetVersion(),
			Usage: &domain.Resource{
				Cpu:    proc.GetUsage().GetCpu(),
				Memory: proc.GetUsage().GetMemory(),
			},
		})
	}

	for _, cont := range p.GetContainers() {
		containers = append(containers, &domain.Service{
			Name:    cont.GetName(),
			Version: cont.GetVersion(),
			Usage: &domain.Resource{
				Cpu:    cont.GetUsage().GetCpu(),
				Memory: cont.GetUsage().GetMemory(),
			},
			Allocation: &domain.Resource{
				Cpu:    cont.GetUsage().GetCpu(),
				Memory: cont.GetUsage().GetMemory(),
			},
		})
	}

	return &domain.Provisioner{
		Id:           p.GetId(),
		Version:      p.GetVersion(),
		Hostname:     p.GetHostname(),
		IpAddress:    p.GetIpAddress(),
		AzName:       p.GetAzName(),
		MachineClass: p.GetMachineClass(),
		Started:      time.Unix(int64(p.GetStarted()), 0),
		LastUpdate:   time.Now(),
		Machine: &domain.Machine{
			Cores:  p.GetMachine().GetCores(),
			Memory: p.GetMachine().GetMemory(),
			Disk:   p.GetMachine().GetDisk(),
			Usage: &domain.Resource{
				Cpu:    p.GetMachine().GetUsage().GetCpu(),
				Memory: p.GetMachine().GetUsage().GetMemory(),
				Disk:   p.GetMachine().GetUsage().GetDisk(),
			},
		},
		Processes:  processes,
		Containers: containers,
	}
}

func SubProvisioningInfo(req *server.Request) (proto.Message, errors.Error) {
	request := &iproto.Info{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest(server.Name+".subprovisioninginfo", err.Error())
	}

	p := protoToProvisioner(request)
	registry.Insert(p)

	return nil, nil
}
