package nsq

import (
	log "github.com/cihub/seelog"
	nsqlib "github.com/HailoOSS/go-nsq"
)

type logBridge struct{}

func (lb logBridge) Output(depth int, s string) error {
	switch s[:3] {
	case nsqlib.LogLevelDebugPrefix:
		log.Debug(s[4:])
	case nsqlib.LogLevelInfoPrefix:
		log.Info(s[4:])
	case nsqlib.LogLevelWarningPrefix:
		log.Warn(s[4:])
	case nsqlib.LogLevelErrorPrefix:
		log.Error(s[4:])
	}
	return nil
}
