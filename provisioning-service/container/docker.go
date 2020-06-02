package container

import (
	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strings"

	log "github.com/cihub/seelog"
	docker "github.com/fsouza/go-dockerclient"

	"github.com/HailoOSS/provisioning-service/info"
)

var (
	envFile     = "/opt/hailo/env.sh"
	registryUrl = "docker-registry-meta.elasticride.com:443"
)

// dockerManager is generic docker client wrapper
type dockerManager struct {
	c *docker.Client
}

func newManager() *dockerManager {
	var dockerEndpoint string
	if envEndpoint := os.Getenv("H2O_DOCKER_ENDPOINT"); envEndpoint != "" {
		dockerEndpoint = envEndpoint
	} else {
		dockerEndpoint = "unix:///var/run/docker.sock"
	}
	log.Infof("Starting docker manager")
	log.Debugf("Docker client endpoint: %v", dockerEndpoint)

	if envRegistryUrl := os.Getenv("H2O_REGISTRY_ENDPOINT"); envRegistryUrl != "" {
		registryUrl = envRegistryUrl
	}
	log.Debugf("Docker registry endpoint: %v", registryUrl)

	client, err := docker.NewClient(dockerEndpoint)
	// We probably want to handle this better here
	if err != nil {
		log.Errorf("Unable to init docker client, endpoint %s: %v", dockerEndpoint, err)
	}

	return &dockerManager{
		c: client,
	}
}

func (m *dockerManager) Download(image, tag string) error {
	return m.c.PullImage(docker.PullImageOptions{
		Repository: registryUrl + "/" + image,
		Tag:        tag,
	}, docker.AuthConfiguration{})
}

func (m *dockerManager) IsDownloaded(image, tag string) bool {
	_, err := m.c.InspectImage(registryUrl + "/" + image + ":" + tag)
	if err != nil {
		log.Warnf("Image %s status: %v", image, err)
		return false
	}
	return true
}

func (m *dockerManager) RemoveContainer(name string) error {
	return m.c.RemoveContainer(docker.RemoveContainerOptions{
		ID:            name,
		RemoveVolumes: true,
		Force:         false,
	})
}

func (m *dockerManager) RemoveImage(name string) error {
	return m.c.RemoveImage(name)
}

// Start starts a container.
func (m *dockerManager) Start(image, tag string, conf *Config) error {
	name := image + "-" + tag
	_, err := m.c.InspectContainer(name)
	if err != nil {
		log.Infof("Creating container %s", name)
		_, err := m.c.CreateContainer(docker.CreateContainerOptions{
			Name: name,
			Config: &docker.Config{
				Env:   getEnv(),
				Image: registryUrl + "/" + image + ":" + tag,
			},
		})

		if err != nil {
			return err
		}
	}
	return m.c.StartContainer(name, &docker.HostConfig{
		Binds:       getBinds(),
		NetworkMode: "host",
	})
}

func (m *dockerManager) InspectContainer(name string) (*docker.Container, error) {
	return m.c.InspectContainer(name)
}

// ListRunning lists services based on a specific filter and further parsing
// FIXME: we really want this to be a proper filter
func (m *dockerManager) ListRunning(filter string) ([]string, error) {
	containers, err := m.c.ListContainers(docker.ListContainersOptions{
		Size: true,
	})
	if err != nil {
		return nil, err
	}

	services := make([]string, 0)
	for _, c := range containers {
		services = append(services, c.Names...)
	}

	// Remove the leading / from the names
	for i := 0; i < len(services); i++ {
		services[i] = services[i][1:]
	}

	rsp := make([]string, 0)
	r, _ := regexp.Compile(filter)
	if filter != "" {
		for _, s := range services {
			if r.Match([]byte(s)) {
				rsp = append(rsp, s)
			}
		}
	} else {
		rsp = services
	}

	return rsp, nil
}

// ListContainers returns all containers
func (m *dockerManager) ListContainers(all bool) ([]docker.APIContainers, error) {
	return m.c.ListContainers(docker.ListContainersOptions{
		Size: true,
		All:  all,
	})
}

func (m *dockerManager) IsRunning(name string) bool {
	rsp, err := m.c.InspectContainer(name)
	if err != nil {
		switch err.(type) {
		case *docker.NoSuchContainer:
			return false
		default:
			log.Errorf("Error checking container %s status: %v", name, err)
			return false
		}
	}
	if !rsp.State.Running {
		return false
	}

	return true
}

func (m *dockerManager) Stop(name string, timeout uint) error {

	log.Infof("Stopping container %s", name)
	return m.c.StopContainer(name, timeout)
}

func (m *dockerManager) Restart(name string) error {

	log.Infof("Restarting container %s", name)
	return m.c.RemoveContainer(docker.RemoveContainerOptions{
		ID:    name,
		Force: true,
	})
}

// getEnv parses our env file and makes sure we pass the right envs to our containers
func getEnv() []string {
	var env []string

	iface := "eth0"

	switch runtime.GOOS {
	case "darwin":
		iface = "en0"
	}
	hostIp := info.GetIpAddress(iface)

	b, err := ioutil.ReadFile(envFile)
	if err != nil {
		return env
	}

	buf := bytes.NewBuffer(b)

	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			break
		}

		arr := strings.Split(line, " ")
		if len(arr) == 2 {
			envr := arr[1][:len(arr[1])-1]
			envr = strings.Replace(envr, "127.0.0.1", hostIp, -1)
			envr = strings.Replace(envr, "localhost", hostIp, -1)
			env = append(env, envr)
		}
	}

	//	env = append(env, getRabbitHostEnv())

	return env
}

// Temp bodge until the PRs are merged in the go-platform layer and puppet
func getRabbitHostEnv() string {
	iface := "eth0"

	switch runtime.GOOS {
	case "darwin":
		iface = "en0"
	}
	hostIp := info.GetIpAddress(iface)

	return "BOXEN_RABBITMQ_URL=amqp://hailo:hailo@" + hostIp + ":5672"
}

// getBinds returns the appropriate volume information
func getBinds() []string {
	return []string{
		"/opt/hailo/login-service:/opt/hailo/login-service:ro",
		"/opt/hailo/etc:/opt/hailo/etc:ro",
		"/etc/h2o:/etc/h2o:ro",
		"/opt/hailo/var/log:/opt/hailo/var/log:rw",
	}
}
