package statusmonitor

import (
	"encoding/json"

	log "github.com/cihub/seelog"
	"github.com/HailoOSS/protobuf/proto"
	ptomb "gopkg.in/tomb.v2"

	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/raven"
	"github.com/HailoOSS/platform/util"
	mp "github.com/HailoOSS/monitoring-service/proto/azstatus"
	"time"
)

const retryDelay = time.Second * 5

type Failure int

var failureToString = map[int]string{
	0: "No failures detected",
	1: "Connectivity Failure",
	2: "Monitoring Failure",
}

func (f *Failure) String() string {
	str, ok := failureToString[int(*f)]
	if !ok {
		return "Not implemented"
	}
	return str
}

const (
	NoFailure           Failure = 0
	ConnectivityFailure Failure = 1
	MonitoringFailure   Failure = 2
)

// StatusMonitor represents the AZ Health Status for this API instance
type StatusMonitor struct {
	ptomb.Tomb
	IsHealthy   bool
	LastChanged time.Time
	AZName      string
	FailureType Failure
	lockHandle  string
}

// newStatusMonitor returns a new StatusMonitor object and kicks off the AZ monitoring goroutine
func NewStatusMonitor() *StatusMonitor {
	azName, err := util.GetAwsAZName()
	if err != nil {
		log.Errorf("[StatusMonitor] Unable to get the local AZ name: %v", err)
		azName = "undefined"
	}

	s := &StatusMonitor{
		IsHealthy:   true,
		LastChanged: time.Now(),
		AZName:      azName,
	}

	// Make sure we start active monitoring only if we run on AWS
	if err == nil {
		s.Go(s.monitor)
	}

	return s
}

// monitor is a control function which checks the AZ health every 5 seconds
func (s *StatusMonitor) monitor() error {
	log.Debugf("[StatusMonitor] Starting the AZ monitor...")
	for {
		// Check our rabbitmq connectivity
		ravenStatus := raven.IsConnected()
		s.interpretRavenStatus(ravenStatus)
		// Only call the monitoring service if we are online
		if ravenStatus {
			azStatus, err := checkAZStatus(s.AZName)
			if err == nil {
				// @TODO we might want to add some more retries here
				s.interpretAZStatus(azStatus)
			}
			// For now we move on if we can't contact the monitoring service.
			// Note that if we have failed over the AZ and subsequently we can't contact
			// the monitoring service we remain in failed state
		}

		select {
		case <-s.Dying():
			return nil
		case <-time.After(retryDelay):
		}
	}

}

// interpretRavenStatus applies failover logic in respect to rabbitmq issues
func (s *StatusMonitor) interpretRavenStatus(isConnected bool) {
	if !isConnected {
		log.Infof("[StatusMonitor] Local instance is unable to connect to rabbitmq")

		s.failoverAZ(ConnectivityFailure)
		return
	}
	// Recover only if we had a connectivity issue previously
	if isConnected && !s.IsHealthy && s.FailureType == ConnectivityFailure {
		log.Infof("[StatusMonitor] Local instance successfully reconnected to rabbitmq")
		s.recoverAZ()
		return
	}
}

// interpretAZStatus applies failover logic in respect to issues reported
// by the monitoring service
func (s *StatusMonitor) interpretAZStatus(azStatus bool) {
	if !azStatus && s.IsHealthy {
		log.Infof("[StatusMonitor] Local AZ %s reported unhealthy by the monitoring service", s.AZName)
		s.failoverAZ(MonitoringFailure)
	}

	if azStatus && !s.IsHealthy {
		log.Infof("[StatusMonitor] Local AZ %s reported healthy by the monitoring service - returning to the ELB resource pool", s.AZName)
		s.recoverAZ()
	}
}

// failoverAZ failoves over our AZ and records the failure type
func (s *StatusMonitor) failoverAZ(failure Failure) {
	var err error

	// Try to get a lock first and mark our AZ unhealthy
	s.lockHandle, err = lock(s.AZName)
	if err != nil {
		log.Debugf("[StatusMonitor] Unable to get a lock : %v", err)
		// Check if there is an existing lock for our AZ
		failedAZ, errr := readLock()
		if errr != nil {
			log.Errorf("[StatusMonitor] Unable to read failed az lock file %v", errr)
			return
		}
		if failedAZ != s.AZName {
			log.Infof("[StatusMonitor] Aborting AZ %v failover - another AZ %v is already marked as unhealthy",
				s.AZName, failedAZ)
			return
		}

	}
	// We can fail over now
	log.Infof("[StatusMonitor] Failing over AZ %s and exiting the elb pool", s.AZName)
	s.IsHealthy = false
	s.LastChanged = time.Now()
	s.FailureType = failure
}

// recoverAZ recovers a failed AZ
func (s *StatusMonitor) recoverAZ() {
	// Make sure the lock is removed on recovery
	if s.lockHandle != "" {
		unlock(s.lockHandle)
		s.lockHandle = ""
	}
	s.LastChanged = time.Now()
	s.IsHealthy = true
	s.FailureType = NoFailure
}

// checkAZStatus polls the monitoring service for AZ health status
func checkAZStatus(az string) (bool, error) {
	monRequest := &mp.Request{AzName: proto.String(az)}
	req, err := client.NewRequest("com.HailoOSS.service.monitoring", "azstatus", monRequest)
	if err != nil {
		log.Errorf("[StatusMonitor] Unable to marshal a monitoring service request: %v", err)
		return false, err
	}

	//We don't want to be retrying this
	reqOptions := make(client.Options)
	reqOptions["retries"] = 0
	req.SetFrom("com.HailoOSS.hailo-2-api")

	rsp, err := client.CustomReq(req, reqOptions)
	if err != nil {
		log.Errorf("[StatusMonitor] Unable to call the monitoring service: %v", err)
		return false, err
	}

	monResponse := &mp.Response{}
	err = rsp.Unmarshal(monResponse)
	if err != nil {
		log.Errorf("[StatusMonitor] Unable to unmarshal the monitoring service response: %v", err)
		return false, err
	}

	return monResponse.GetIsHealthy(), nil

}

// JsonResponse represents a generic json object
type JsonResponse map[string]interface{}

// Json Marshaling
func (r JsonResponse) String() (s string) {
	b, err := json.Marshal(r)
	if err != nil {
		s = ""
		return
	}
	s = string(b)
	return
}
