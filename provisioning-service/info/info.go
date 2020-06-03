package info

import (
	"fmt"
	"net"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	log "github.com/cihub/seelog"
	sigar "github.com/cloudfoundry/gosigar"
	"github.com/hailo-platform/H2O/platform/client"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/platform/util"
	"github.com/hailo-platform/H2O/protobuf/proto"
	"github.com/hailo-platform/H2O/provisioning-service/dao"
	iproto "github.com/hailo-platform/H2O/provisioning-service/proto"
)

const (
	updateInterval = time.Second * 20
)

var (
	hostname     string
	azName       string
	machineClass string
	version      string
	ipAddress    string
	started      uint64
	numCpu       uint64
	cpuSample    *sigar.Cpu
	procSample   map[string]*proc
)

func init() {
	hostname, _ = os.Hostname()
	azName, _ = util.GetAwsAZName()
	machineClass = os.Getenv("H2O_MACHINE_CLASS")
	if len(machineClass) == 0 {
		machineClass = "default"
	}

	iface := "eth0"

	switch runtime.GOOS {
	case "darwin":
		iface = "en1"
	}

	ipAddress = GetIpAddress(iface)
	started = uint64(time.Now().Unix())
	numCpu = uint64(runtime.NumCPU())
	cpuSample, _ = getCpu()
	procSample, _ = getProcUsage()
}

func GetIpAddress(iface string) string {
	var ipAddress string

	i, _ := net.InterfaceByName(iface)
	addrs, _ := i.Addrs()

	for _, addr := range addrs {
		if strings.Contains(addr.String(), ":") {
			continue
		}

		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}

		ipAddress = ip.String()
		break
	}

	return ipAddress
}

func getMachineInfo(cpu sigar.Cpu) (*iproto.Machine, error) {
	cpuUsed := getCpuUsage(cpu)
	memory, _ := getMemory()
	disk, _ := getDisk()

	return &iproto.Machine{
		Cores:  proto.Uint64(numCpu),
		Memory: proto.Uint64(memory.Total),
		Disk:   proto.Uint64(disk.Total),
		Usage: &iproto.Resource{
			Cpu:    proto.Float64(cpuUsed),
			Memory: proto.Uint64(memory.ActualUsed),
			Disk:   proto.Uint64(disk.Used),
		},
	}, nil
}

func getNameVersion(process string) (string, string) {
	_, p := path.Split(process)
	last := strings.LastIndex(p, "-")
	if last == -1 {
		return p, ""
	}
	return p[:last], p[last+1:]
}

func getServices(cpu sigar.Cpu) (map[string][]*iproto.Service, error) {
	procs, err := getProcUsage()
	if err != nil {
		return nil, err
	}

	types := make(map[string]dao.ServiceType)
	processes := make(map[string][]*iproto.Service)
	services, _ := dao.CachedServices(machineClass)

	for _, service := range services {
		key := fmt.Sprintf("%s-%d", service.ServiceName, service.ServiceVersion)
		types[key] = service.ServiceType
	}

	for proc, u := range procs {
		usage := &iproto.Resource{}
		if ou, ok := procSample[proc]; ok {
			usage.Cpu = proto.Float64(getProcCpuUsage(*u.cpu, *ou.cpu, cpu.Total()))
		}
		usage.Memory = proto.Uint64(u.mem.Resident)
		name, version := getNameVersion(proc)

		process := &iproto.Service{
			Name:    proto.String(name),
			Version: proto.String(version),
			Usage:   usage,
		}

		var typ string
		key := fmt.Sprintf("%s-%s", name, version)
		switch types[key] {
		case dao.ServiceTypeContainer:
			typ = "container"
		default:
			typ = "process"
		}

		processes[typ] = append(processes[typ], process)
	}

	procSample = procs
	return processes, nil
}

func pubInfo() error {
	cpu, _ := getCpu()
	delta := (*cpu).Delta(*cpuSample)
	cpuSample = cpu
	services, _ := getServices(delta)
	machineInfo, _ := getMachineInfo(delta)

	return client.Pub("com.hailo-platform/H2O.kernel.provisioning.info", &iproto.Info{
		Id:           proto.String(server.InstanceID),
		Version:      proto.String(version),
		Hostname:     proto.String(hostname),
		IpAddress:    proto.String(ipAddress),
		AzName:       proto.String(azName),
		MachineClass: proto.String(machineClass),
		Started:      proto.Uint64(started),
		Timestamp:    proto.Uint64(uint64(time.Now().Unix())),
		Machine:      machineInfo,
		Processes:    services["process"],
		Containers:   services["container"],
	})
}

func run() {
	version = strconv.FormatUint(server.Version, 10)
	ticker := time.NewTicker(updateInterval)
	for {
		select {
		case <-ticker.C:
			if err := pubInfo(); err != nil {
				log.Errorf("Error publishing info: %v", err)
			}
		}
	}
}

func Run() {
	go run()
}
