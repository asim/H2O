package dao

import (
	"github.com/HailoOSS/protobuf/proto"
	pproto "github.com/HailoOSS/provisioning-manager-service/proto/provisioned"
	"strconv"
)

type RunLevel struct {
	Region string
	Level  int64
}

type ServiceRunLevels struct {
	ServiceName string
	Levels      [6]bool
}

type Service struct {
	Id              string `cf:"provisioned_service" key:"Id"`
	ServiceName     string `name:"servicename"`
	ServiceVersion  uint64 `name:"serviceversion"`
	MachineClass    string `name:"machineclass"`
	NoFileSoftLimit uint64 `name:"nofilesoftlimit"`
	NoFileHardLimit uint64 `name:"nofilehardlimit"`
	ServiceType     uint64 `name:"servicetype"`
}

type storedRunLevel struct {
	Id     string `cf:"run_levels" key:"Id"`
	Region string `name:"region"`
	Level  int64  `name:"level"`
}

type storedServiceRunLevels struct {
	Id          string `cf:"service_run_levels" key:"Id"`
	ServiceName string `name:"servicename"`
	Levels      string `name:"levels"`
}

func (r *RunLevel) id() string {
	return regionPrefix + r.Region
}

func (r *RunLevel) stored() *storedRunLevel {
	return &storedRunLevel{
		Id:     r.id(),
		Region: r.Region,
		Level:  r.Level,
	}
}

func (sr *ServiceRunLevels) id() string {
	return sr.ServiceName
}

func (sr *ServiceRunLevels) stored() *storedServiceRunLevels {
	return &storedServiceRunLevels{
		Id:          sr.id(),
		ServiceName: sr.ServiceName,
		Levels:      flatten(sr.Levels),
	}
}

func (s *storedRunLevel) runLevel() *RunLevel {
	return &RunLevel{
		Region: s.Region,
		Level:  s.Level,
	}
}

func (s *storedServiceRunLevels) runLevels() *ServiceRunLevels {
	return &ServiceRunLevels{
		ServiceName: s.ServiceName,
		Levels:      unflattenServiceRunLevels(s.Levels),
	}
}

func (s *Service) id() string {
	return generateHash(s.ServiceName, strconv.FormatUint(s.ServiceVersion, 10), s.MachineClass)
}

func (s *Service) ToProto() *pproto.Service {
	return &pproto.Service{
		ServiceName:     proto.String(s.ServiceName),
		ServiceVersion:  proto.Uint64(s.ServiceVersion),
		MachineClass:    proto.String(s.MachineClass),
		NoFileSoftLimit: proto.Uint64(s.NoFileSoftLimit),
		NoFileHardLimit: proto.Uint64(s.NoFileHardLimit),
		ServiceType:     proto.Uint64(s.ServiceType),
	}
}
