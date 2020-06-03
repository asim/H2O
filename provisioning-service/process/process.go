package process

import (
	"bytes"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/platform/util"
	dao "github.com/hailo-platform/H2O/provisioning-service/dao"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	osName       = runtime.GOOS
	defaultUser  = "hailosvc"
	defaultGroup = "hailosvc"
)

var (
	initCtl = newInitCtler()
	exeDir  = "/opt/hailo/bin" // where we store downloaded executable files
)

type initCtler interface {
	List(string) ([]string, error)
	Install(string, uint64, uint64, uint64) error
	Start(string, uint64, uint64, uint64) error
	Stop(string, uint64) error
	Restart(string, uint64) error
	Uninstall(string, uint64) error
}

type config struct {
	Directory string
	Extension string
}

type platform struct {
	InitCmd string
	Config  config
}

func combineNameVersion(serviceName string, serviceVersion uint64) string {
	return serviceName + "-" + strconv.Itoa(int(serviceVersion))
}

func getConfDir() string {
	if dir := os.Getenv("HAILO_INIT_DIR"); dir != "" {
		return dir
	}

	return defaultConfDir
}

func getConfPath(serviceName string, serviceVersion uint64, conf config) string {
	return path.Join(conf.Directory, combineNameVersion(serviceName, serviceVersion)+conf.Extension)
}

func getEnvironment() map[string]string {
	env := make(map[string]string)
	for _, val := range os.Environ() {
		vals := strings.SplitN(val, "=", 2)
		env[vals[0]] = vals[1]
	}
	return env
}

// getExePath() returns a string containing the path for this provisioned
// service's executable, once downloaded to the local filesystem.
func getExePath(serviceName string, serviceVersion uint64) string {
	return path.Join(exeDir, combineNameVersion(serviceName, serviceVersion))
}

// Returns an interface capable of executing the local OS init cmd.
func newInitCtler() initCtler {
	return newPlatform()
}

// run executes any given command
func run(cmdName string, args ...string) error {
	cmd := exec.Command(cmdName, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v; stdout: %q, stderr: %q", err, stdout.Bytes(), stderr.Bytes())
	}

	return nil
}

// convenience function which wraps getExePath()
func ExePath(ps *dao.ProvisionedService) string {
	return getExePath(ps.ServiceName, ps.ServiceVersion)
}

func Install(serviceName string, serviceVersion, noFileSoftLimit, noFileHardLimit uint64) error {
	return initCtl.Install(serviceName, serviceVersion, noFileSoftLimit, noFileHardLimit)
}

func Uninstall(serviceName string, serviceVersion uint64) error {
	return initCtl.Uninstall(serviceName, serviceVersion)
}

func Start(serviceName string, serviceVersion, noFileSoftLimit, noFileHardLimit uint64) error {
	return initCtl.Start(serviceName, serviceVersion, noFileSoftLimit, noFileHardLimit)
}

func Stop(serviceName string, serviceVersion uint64) error {
	return initCtl.Stop(serviceName, serviceVersion)
}

func RestartAZ(azName string) error {
	thisAzName, err := util.GetAwsAZName()
	if err != nil {
		log.Errorf("Unable to determine this az name. %s", err)
		return err
	}
	if azName != thisAzName {
		// skip
		return nil
	}
	provisionedServices, err := dao.CachedServices(getMachineClass())
	if err != nil {
		return fmt.Errorf("Error restarting AZ. Could not retrieve list of provisioned services. %s", err)
	}
	var lastErr error
	for _, p := range provisionedServices {
		time.Sleep(time.Duration(rand.Int63n(5)) * time.Second) // jitter 5 second to reduce thundering herd
		if thisErr := initCtl.Restart(p.ServiceName, p.ServiceVersion); thisErr != nil {
			log.Errorf("Error restarting service %s", thisErr)
			lastErr = thisErr
		}
	}
	log.Critical("Restart AZ finished. Committing suicide.")
	os.Exit(0)
	return lastErr
}

func Restart(serviceName string, serviceVersion uint64, azName string) error {
	thisAzName, err := util.GetAwsAZName()
	if err != nil {
		log.Errorf("Unable to determine this az name")
		return err
	}
	if len(azName) > 0 && thisAzName != azName {
		// skip
		return nil
	}
	// add some random jitter 0-60 seconds
	time.Sleep(time.Duration(rand.Int63n(60)) * time.Second)
	return initCtl.Restart(serviceName, serviceVersion)
}

func ListRunning(matching string) ([]string, error) {
	return initCtl.List(matching)
}

func CachedCountRunningInstances(serviceName string, serviceVersion uint64, processes []string) int {
	instance := combineNameVersion(serviceName, serviceVersion)
	running := 0
	for _, process := range processes {
		if instance == process {
			running++
		}
	}
	return running
}

func CountRunningInstances(serviceName string, serviceVersion uint64) (int, error) {
	processes, err := ListRunning(combineNameVersion(serviceName, serviceVersion))
	if err != nil {
		return -1, err
	}
	return len(processes), nil
}

func install(serviceName string, serviceVersion, noFileSoftLimit, noFileHardLimit uint64, conf config, tmpl *template.Template) error {

	cmdName := combineNameVersion(serviceName, serviceVersion)
	exePath := getExePath(serviceName, serviceVersion)
	confPath := getConfPath(serviceName, serviceVersion, conf)

	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("Unable to install, binary does not exist: %v", exePath)
	}

	if err := os.MkdirAll(conf.Directory, 0777); err != nil {
		return err
	}

	_, err := os.Stat(confPath)
	if err == nil {
		log.Info("Deleting existing init file: ", confPath)
		os.Remove(confPath)
	}

	if err := os.MkdirAll(exeDir, 0777); err != nil {
		return err
	}

	file, err := os.Create(confPath)
	if err != nil {
		return err
	}
	defer file.Close()

	user := os.Getenv("HAILO_INIT_RUNASUSER")
	group := os.Getenv("HAILO_INIT_RUNASGROUP")

	if user == "" {
		user = defaultUser
		log.Info("HAILO_INIT_RUNASUSER was not set - defaulting to " + user)
	}

	if group == "" {
		group = defaultGroup
		log.Info("HAILO_INIT_RUNASGROUP was not set - defaulting to " + group)
	}

	if noFileSoftLimit < 1024 {
		noFileSoftLimit = 1024
	}

	if noFileHardLimit < 1024 {
		noFileHardLimit = 4096
	}

	params := &struct {
		Description     string
		Author          string
		ProcessName     string
		GeneratedAt     string
		RunAsUser       string
		RunAsGroup      string
		NoFileSoftLimit string
		NoFileHardLimit string
		Environment     map[string]string
	}{
		cmdName,
		"com.hailo-platform/H2O.service.provisioning.create",
		exePath,
		time.Now().String(),
		user,
		group,
		strconv.Itoa(int(noFileSoftLimit)),
		strconv.Itoa(int(noFileHardLimit)),
		getEnvironment(),
	}

	err = tmpl.Execute(file, params)
	if err != nil {
		return err
	}
	return nil
}

func uninstall(serviceName string, serviceVersion uint64, conf config) error {
	confPath := getConfPath(serviceName, serviceVersion, conf)
	if err := os.Remove(confPath); err != nil {
		return err
	}

	return nil
}

func getMachineClass() string {
	myClass := os.Getenv("H2O_MACHINE_CLASS")
	if len(myClass) == 0 {
		myClass = "default"
	}
	return myClass
}
