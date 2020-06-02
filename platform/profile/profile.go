package profile

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"sync"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/platform/util"
	"github.com/HailoOSS/service/config"
	"github.com/jono-macd/profile"
)

var (
	defaultS3Path = "pprof-data"
	binaryPath    = "/opt/hailo/bin/"
)

type Output int

const (
	S3Output Output = iota
	FileOutput
)

type Store interface {
	Save(id string, reader io.Reader, contentType string) (path string, err error)
}

type RunningProfile struct {
	sync.Mutex
	Prof interface {
		Stop()
	}
	Store       Store
	Buf         *bufferCloser
	ProfileType string
	Running     bool
	Id          string
	Name        string
	Bucket      string
}

type bufferCloser struct {
	bytes.Buffer
}

func (b *bufferCloser) Close() error { return nil }

var currentProfileRun *RunningProfile

func init() {
	currentProfileRun = &RunningProfile{}
	currentProfileRun.Running = false

	defaultS3Path = config.AtPath("hailo", "platform", "profile", "defaultS3Path").AsString("pprof-data")
	binaryPath = config.AtPath("hailo", "platform", "profile", "binaryPath").AsString("/opt/hailo/bin")
}

// StartProfiling will start up profiling with the given config
func StartProfiling(cfg *profile.Config, output Output, name, id string) error {
	currentProfileRun.Lock()
	defer currentProfileRun.Unlock()

	// Only start if there is no other profile running
	if currentProfileRun.Running {
		return fmt.Errorf("Profile already started")
	}

	// Create a buffer for the results
	currentProfileRun.Buf = &bufferCloser{}
	cfg.OverrideWriter = currentProfileRun.Buf

	if cfg.BlockProfile {
		currentProfileRun.ProfileType = "block"
	}
	if cfg.CPUProfile {
		currentProfileRun.ProfileType = "cpu"
	}
	if cfg.MemProfile {
		currentProfileRun.ProfileType = "memory"
	}

	// Set some defaults
	cfg.NoShutdownHook = true

	// Start profiling
	currentProfileRun.Prof = profile.Start(cfg)

	// Set up output file path
	path := fmt.Sprintf("%s-%s-%s", util.GetEnvironmentName(), name, id)

	// Set up store
	var (
		as  Store
		err error
	)
	switch output {
	case S3Output:
		log.Debugf("[PProf] Outputting to S3 directory: %s", path)
		path = defaultS3Path + "/" + path
		as, err = NewS3Store(path)
	case FileOutput:
		log.Debugf("[PProf] Outputting to temp directory")
		as, err = NewFileStore()
	}
	if err != nil {
		currentProfileRun.Prof.Stop()
		return fmt.Errorf("Unable to create new store: %v", err)
	}

	currentProfileRun.Store = as
	currentProfileRun.Running = true
	currentProfileRun.Id = id
	currentProfileRun.Name = name
	log.Infof("[PProf] Started profiling run %s for %s", id, name)

	return nil
}

// StopProfiling will stop profiling
func StopProfiling() (string, string, string, error) {

	currentProfileRun.Lock()
	defer currentProfileRun.Unlock()

	// Check if profile is actually running
	if !currentProfileRun.Running {
		return "", "", "", fmt.Errorf("No Profile Running")
	}

	id := currentProfileRun.Id

	// Stop profiling
	currentProfileRun.Prof.Stop()
	currentProfileRun.Running = false

	profileOutput, err := currentProfileRun.Store.Save(currentProfileRun.ProfileType+".pprof", currentProfileRun.Buf, "application/octet-stream")
	if err != nil {
		log.Debugf("[PProf] Cannot save profile: %v", err)
		return id, "", "", fmt.Errorf("Unable to save profile: %v", err)
	}
	log.Infof("[PProf] Saved profile output to: %s", profileOutput)

	binaryLocation := fmt.Sprintf("%s/%s", binaryPath, currentProfileRun.Name)
	f, err := os.Open(binaryLocation)
	defer f.Close()
	if err != nil {
		log.Debugf("[PProf] Cannot read binary %s : %v", binaryLocation, err)
		return id, "", "", fmt.Errorf("Unable to read binary: %v", err)
	}

	binaryOutput, err := currentProfileRun.Store.Save(currentProfileRun.Name, f, "application/octet-stream")
	if err != nil {
		log.Debugf("[PProf] Cannot save binary: %v", err)
		return id, "", "", fmt.Errorf("Unable to save to store: %v", err)
	}

	log.Infof("[PProf] Saved binary to: %s", binaryOutput)

	return id, profileOutput, binaryOutput, nil
}
