package deps

import (
	"encoding/json"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/provisioning-service/dao"
	"github.com/HailoOSS/provisioning-service/pkgmgr"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultPrefix = "hailo-deps"
)

var (
	checkInterval  = 120
	defaultManager = newManager()
	myClass        = os.Getenv("H2O_MACHINE_CLASS")
	prefix         = os.Getenv("HAILO_DEPS_BUCKET")
)

type depsManager struct {
	mtx sync.Mutex
}

func init() {
	if len(prefix) == 0 {
		prefix = defaultPrefix
	}

	if len(myClass) == 0 {
		myClass = "default"
	}
}

func newManager() *depsManager {
	return &depsManager{}
}

// getFileList retrieves a list of dependency files from the config service.
func getFileList(serviceName string) []map[string]string {
	service := strings.Replace(serviceName, ".", "-", -1)
	jlist := config.AtPath("hailo", "dependencies", service, "files").AsJson()

	var list []map[string]string
	json.Unmarshal(jlist, &list)
	return list
}

// load checks whether a file exists in S3 then downloads the file.
func load(remotePath, localPath string) error {
	if _, err := pkgmgr.DownloadFile(prefix, remotePath, localPath); err != nil {
		return err
	}

	if err := os.Chmod(localPath, 0644); err != nil {
		return err
	}

	return nil
}

// run periodically checks for missing dependencies and loads them.
func (m *depsManager) run() {
	ticker := time.NewTicker(time.Duration(checkInterval) * time.Second)

	for {
		select {
		case <-ticker.C:
			services, err := dao.CachedServices(myClass)
			if err != nil {
				log.Errorf("[deps] Error retrieving provisioned services list: %v", err)
				continue
			}

			for _, s := range services {
				m.load(s.ServiceName)
			}
		}
	}
}

// load installs the dependencies from the remote path.
func (m *depsManager) load(serviceName string) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	files := getFileList(serviceName)
	if len(files) == 0 {
		log.Debugf("No dependencies to load for service %s", serviceName)
		return nil
	}

	log.Debugf("Loading dependencies for service %s", serviceName)
	for _, file := range files {
		localPath, remotePath := file["localpath"], file["remotepath"]
		if _, err := os.Stat(localPath); !os.IsNotExist(err) {
			// If the file exists we just skip it.
			continue
		}

		err := load(remotePath, localPath)
		if err != nil {
			return err
		}
	}

	return nil
}

// Load installs dependencies for a service.
func Load(serviceName string) error {
	return defaultManager.load(serviceName)
}

// Run starts the deps runner
func Run() {
	go defaultManager.run()
}
