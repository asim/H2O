package testing

import (
	"fmt"
	"io/ioutil"
	golog "log"
	"os"
	"sync"
	"testing"

	log "github.com/cihub/seelog"
	"github.com/stretchr/testify/require"
	testifysuite "github.com/stretchr/testify/suite"
)

const testLoggerConfigTemplate = `<seelog type="sync" minlevel="%s">
    <outputs formatid="main">
        <custom name="%s" formatid="main"/>
    </outputs>
    <formats>
        <format id="main" format="%%Date %%Time [%%LEV] %%Msg (%%File %%Line)%%n"/>
    </formats>
</seelog>`

var (
	loggerIdNum = 0

	// Mutex which ensures only one suite is run at a time
	suiteRunnerMtx = &sync.Mutex{}
)

// InstallTestLogger will install a logger with appropriate configuration for the current testing setup. It returns a
// the logger that was overwritten; this should be restored (via a call to RestoreLogger) at the conclusion of the test.
func InstallTestLoggerWithLogLevel(t testing.TB, logLevel string) log.LoggerInterface {
	golog.SetOutput(ioutil.Discard)

	loggerIdNum++
	loggerId := fmt.Sprintf("testLogger%d", loggerIdNum)

	log.RegisterReceiver(loggerId, &testLogger{tb: t})

	config := fmt.Sprintf(testLoggerConfigTemplate, logLevel, loggerId)
	if logger, err := log.LoggerFromConfigAsString(config); err != nil {
		panic(err)
	} else {
		stashed := log.Current
		log.ReplaceLogger(logger)
		return stashed
	}
}

func InstallTestLogger(t testing.TB) log.LoggerInterface {
	logLevel := "trace"
	if !testing.Verbose() {
		logLevel = "warn"
	}
	return InstallTestLoggerWithLogLevel(t, logLevel)
}

func RestoreLogger(logger log.LoggerInterface) {
	golog.SetOutput(os.Stderr)
	log.ReplaceLogger(logger)
}

// Suite is a basic testing suite with methods for storing and retrieving the current *testing.T context. It also sets
// logging levels appropriately for each test (silencing anything lower than WARN level)
type Suite struct {
	testifysuite.Suite
	*require.Assertions

	LogLevel    string
	loggerLock  sync.Mutex
	loggerStack []log.LoggerInterface
}

func (s *Suite) SetupSuite() {
	s.loggerLock.Lock()
	defer s.loggerLock.Unlock()

	if s.LogLevel == "" {
		s.LogLevel = "trace"
		if !testing.Verbose() {
			s.LogLevel = "warn"
		}
	}

	stash := InstallTestLoggerWithLogLevel(s.T(), s.LogLevel)
	s.loggerStack = []log.LoggerInterface{stash}
	s.Assertions = require.New(s.T())
}

func (s *Suite) TearDownSuite() {
	s.loggerLock.Lock()
	defer s.loggerLock.Unlock()

	if len(s.loggerStack) > 0 {
		RestoreLogger(s.loggerStack[0])
		s.loggerStack = nil
	}
}

func (s *Suite) SetupTest() {
	s.loggerLock.Lock()
	defer s.loggerLock.Unlock()

	stash := InstallTestLogger(s.T())
	s.loggerStack = append(s.loggerStack, stash)

	s.Assertions = require.New(s.T())
}

func (s *Suite) TearDownTest() {
	s.loggerLock.Lock()
	defer s.loggerLock.Unlock()

	if len(s.loggerStack) > 0 {
		RestoreLogger(s.loggerStack[len(s.loggerStack)-1])
		s.loggerStack = s.loggerStack[:len(s.loggerStack)-1]
	}
}

func RunSuite(t *testing.T, suite testifysuite.TestingSuite) {
	suiteRunnerMtx.Lock()
	defer suiteRunnerMtx.Unlock()
	testifysuite.Run(t, suite)
}
