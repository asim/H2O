package info

import (
	"math"
	"strings"

	sigar "github.com/cloudfoundry/gosigar"
)

const (
	ticks    = 100
	diskPath = "/opt/hailo"
)

type proc struct {
	cpu *sigar.ProcTime
	mem *sigar.ProcMem
}

func roundFloat(x float64, prec int) float64 {
	var rounder float64
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)
	intermed += .5
	x = .5
	if frac < 0.0 {
		x = -.5
		intermed -= 1
	}
	if frac >= x {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}
	return rounder / pow
}

func addProcCpu(cpu sigar.ProcTime, other sigar.ProcTime) sigar.ProcTime {
	return sigar.ProcTime{
		StartTime: cpu.StartTime,
		User:      cpu.User + other.User,
		Sys:       cpu.Sys + other.Sys,
		Total:     cpu.Total + other.Total,
	}
}

func addProcMem(mem sigar.ProcMem, other sigar.ProcMem) sigar.ProcMem {
	return sigar.ProcMem{
		Size:        mem.Size + other.Size,
		Resident:    mem.Resident + other.Resident,
		Share:       mem.Share + other.Share,
		MinorFaults: mem.MinorFaults + other.MinorFaults,
		MajorFaults: mem.MajorFaults + other.MajorFaults,
		PageFaults:  mem.PageFaults + other.PageFaults,
	}
}

func getCpu() (*sigar.Cpu, error) {
	cpu := &sigar.Cpu{}
	err := cpu.Get()
	if err != nil {
		return cpu, err
	}

	cpu.User *= (1000 / ticks)
	cpu.Sys *= (1000 / ticks)
	cpu.Nice *= (1000 / ticks)
	cpu.Idle *= (1000 / ticks)
	cpu.Wait *= (1000 / ticks)
	cpu.Irq *= (1000 / ticks)
	cpu.SoftIrq *= (1000 / ticks)
	cpu.Stolen *= (1000 / ticks)
	return cpu, err
}

func getDisk() (*sigar.FileSystemUsage, error) {
	disk := &sigar.FileSystemUsage{}
	err := disk.Get(diskPath)
	return disk, err
}

func getMemory() (*sigar.Mem, error) {
	mem := &sigar.Mem{}
	err := mem.Get()
	return mem, err
}

func getCpuUsage(cpu sigar.Cpu) float64 {
	used := cpu.User + cpu.Sys + cpu.Nice + cpu.Irq + cpu.SoftIrq + cpu.Stolen
	idle := cpu.Idle + cpu.Wait
	return roundFloat(float64(used)/float64(used+idle), 4)
}

func getProcCpuUsage(c sigar.ProcTime, oc sigar.ProcTime, total uint64) float64 {
	used := (c.User + c.Sys) - (oc.User + oc.Sys)
	return roundFloat(float64(numCpu*used)/float64(total), 4)
}

func getProcUsage() (map[string]*proc, error) {
	procs := make(map[string]*proc)
	ps, err := getProcessList()
	if err != nil {
		return procs, err
	}

	for pc, pids := range ps {
		procs[pc] = totalProcUsage(pids)
	}

	return procs, nil
}

func getProcessList() (map[string][]int, error) {
	pl := &sigar.ProcList{}
	if err := pl.Get(); err != nil {
		return nil, err
	}

	procs := make(map[string][]int)
	cpids := make(map[int][]int)

	// iterate over all the pids in /proc/
	for _, pid := range pl.List {
		status := &sigar.ProcState{}
		if err := status.Get(pid); err != nil {
			continue
		}

		// is it a com.hailocab process?
		if !strings.Contains(status.Name, "com.hailocab") {
			if status.Ppid > 1 {
				// hold onto pid as it may be a child of one we know about
				cpids[status.Ppid] = append(cpids[status.Ppid], pid)
			}
			continue
		}

		// /proc/[pid]/status must now already contain Name: com.hailocab
		proc := &sigar.ProcArgs{}
		if err := proc.Get(pid); err != nil {
			continue
		}

		for _, arg := range proc.List {
			if strings.Contains(arg, "com.hailocab") {
				procs[arg] = append(procs[arg], pid)
				break
			}
		}
	}

	// append child pids
	for name, ppids := range procs {
		if cpds, ok := cpids[ppids[0]]; ok {
			ppids = append(ppids, cpds...)
			procs[name] = ppids
		}
	}

	return procs, nil
}

func totalProcUsage(pids []int) *proc {
	tcpu := sigar.ProcTime{}
	tmem := sigar.ProcMem{}

	for _, pid := range pids {
		cpu := &sigar.ProcTime{}
		if err := cpu.Get(pid); err != nil {
			continue
		}
		mem := &sigar.ProcMem{}
		if err := mem.Get(pid); err != nil {
			continue
		}
		tcpu = addProcCpu(tcpu, *cpu)
		tmem = addProcMem(tmem, *mem)
	}

	return &proc{
		cpu: &tcpu,
		mem: &tmem,
	}
}
