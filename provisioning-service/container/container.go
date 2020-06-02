package container

import (
	docker "github.com/fsouza/go-dockerclient"
)

var (
	manager ContainerManager
)

func init() {
	manager = newManager()
}

// Port is a port mapping for a single container
type Port struct {
	Id            string
	HostPort      int
	ContainerPort int
	Protocol      string
	HostIP        string
}

type Volume struct {
	Id         string
	ReadOnly   bool
	MounthPath string
}

// Container Related Config
// Could be abstracted into an interface or moved to the docker class
type Config struct {
	Image   string
	Command []string
	// Defaults to unlimited.
	Memory int
	// Defaults to unlimited.
	CPU     int
	Ports   []Port
	Volumes []Volume
}

// Manager abstracts the container interface
type ContainerManager interface {
	Start(image, tag string, config *Config) error
	Stop(name string, timeout uint) error
	Download(image, tag string) error
	IsDownloaded(image, tag string) bool
	ListRunning(filter string) ([]string, error)
	ListContainers(all bool) ([]docker.APIContainers, error)
	IsRunning(name string) bool
	RemoveContainer(name string) error
	RemoveImage(name string) error
	InspectContainer(name string) (*docker.Container, error)
}

func Start(image, tag string, config *Config) error {
	return manager.Start(image, tag, config)
}

func Stop(name string, timeout uint) error {
	return manager.Stop(name, timeout)
}

func Download(image, tag string) error {
	return manager.Download(image, tag)
}

func IsDownloaded(image, tag string) bool {
	return manager.IsDownloaded(image, tag)
}

func ListRunning(filter string) ([]string, error) {
	return manager.ListRunning(filter)
}

func IsRunning(name string) bool {
	return manager.IsRunning(name)
}

func InspectContainer(name string) (*docker.Container, error) {
	return manager.InspectContainer(name)
}

func RemoveImage(name string) error {
	return manager.RemoveImage(name)
}

func RemoveContainer(name string) error {
	return manager.RemoveContainer(name)
}

func ListContainers(all bool) ([]docker.APIContainers, error) {
	return manager.ListContainers(all)
}
