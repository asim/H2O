package logs

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	glob "github.com/obeattie/ohmyglob"
)

const (
	accessLogEnv        = "H20_ACCESS_LOG_DIR"
	traceLoggingTimeout = time.Second * 60
	traceLoggingLevel   = `<seelog minlevel="trace">
	    <outputs formatid="main">
	        <console/>
	    </outputs>
	    <formats>
	        <format id="main" format="%Date %Time [%LEV] %Msg (%File %Line)%n"/>
	    </formats>
	</seelog>`
)

var (
	configDir  string
	configFile string

	mu sync.Mutex
)

func init() {
	// get boxen AMQP config from environment
	if configDir = os.Getenv("BOXEN_CONFIG_DIR"); configDir == "" {
		configDir = "/opt/hailo/etc"
	}

	configFile = filepath.Join(configDir, "seelog.xml")
	loadLogConfig(configFile)
}

func LoadServiceConfig(name string) {
	servicesConfigFile := configDir + "/" + name + "-seelog.xml"
	if _, err := os.Stat(servicesConfigFile); err == nil {
		loadLogConfig(servicesConfigFile)
	}
}

func CreateAccessLogger(name string) io.WriteCloser {
	// Try and open the access log file
	if accessLogDir := os.Getenv(accessLogEnv); accessLogDir != "" {
		accessLogFilename := filepath.Join(accessLogDir, name+"-access.log")
		commonLogger, err := os.OpenFile(accessLogFilename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.Errorf("[Logs] Error opening access log file: %v", err)
			return nil
		}

		return commonLogger
	} else {
		log.Tracef("[Logs] No access log env (%v) set", accessLogEnv)
		return nil
	}
}

func EnableTrace() {
	mu.Lock()
	defer mu.Unlock()

	newLogger, err := log.LoggerFromConfigAsString(traceLoggingLevel)
	if err != nil {
		log.Warnf("[Service] Unable to construct trace logger: %s", err.Error())
		return
	}
	log.ReplaceLogger(newLogger)
	log.Tracef("[Service] Enabled trace logging for %s", traceLoggingTimeout.String())
	time.Sleep(traceLoggingTimeout)
	log.Tracef("[Service] Reverting to previous logger")
	loadLogConfig(configFile)
}

func loadLogConfig(configFile string) {
	// check for custom logging
	if logger, err := log.LoggerFromConfigAsFile(configFile); err != nil {
		log.Errorf("[Server] Error loading custom logging config from %s: %v", configFile, err)
	} else {
		log.ReplaceLogger(logger)
		glob.Logger = logger
		log.Infof("[Server] Custom logging enabled from %s", configFile)
	}
}
