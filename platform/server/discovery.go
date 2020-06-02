package server

import (
	"os"
	"sync"
	"time"

	log "github.com/cihub/seelog"
	dscShared "github.com/HailoOSS/discovery-service/proto"
	register "github.com/HailoOSS/discovery-service/proto/register"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/util"
	"github.com/HailoOSS/service/auth"
	"github.com/HailoOSS/protobuf/proto"
)

const (
	lostContactInterval  = 60 * time.Second
	tryDiscoveryInterval = 10 * time.Second
	maxDisconnects       = 5
)

type discovery struct {
	sync.RWMutex
	id        string
	reg       *registry
	hostname  string
	connected bool
	hb        *heartbeat
	// isMultiRegistered will be set to true if we believe we're registered with the discovery service. If this is
	// false, we won't wait for a heartbeat timeout to try multiregistering again
	isMultiRegistered bool
}

func newDiscovery(opts *Options) *discovery {
	d := &discovery{
		hostname:  hostname,
		connected: false,
		hb:        newHeartbeat(lostContactInterval),
	}

	go d.tick(opts.Die)

	return d
}

func (self *discovery) tick(die bool) {
	failCount := 0

	ticker := time.NewTicker(tryDiscoveryInterval)

	for {
		select {
		case <-ticker.C:
			self.Lock()
			if !self.isMultiRegistered || !self.hb.healthy() {
				failCount++
				log.Infof("[Server] Service has not received heartbeats within %v and is now disconnected", lostContactInterval)

				if failCount >= maxDisconnects && die {
					log.Criticalf("[Service] Max disconnects (%d) reached, bye bye cruel world", maxDisconnects)
					cleanupLogs()
					os.Exit(1)
				}

				self.connected = false
				self.Unlock()
				if err := self.connect(); err == nil {
					// Successful connection = back to zero
					failCount = 0
				}
			} else {
				self.Unlock()
			}
		}
	}
}

// connect our service/endpoints to the hive mind
// send discovery information on each endpoint to the discovery service
// if successful, marks us as connected
func (self *discovery) connect() error {

	self.Lock()
	defer self.Unlock()

	log.Trace("[Discovery] Connecting")
	if err := self.callDiscoveryService("multiregister", true); err != nil {
		self.isMultiRegistered = false
		return err
	}

	// now connected - set auth scope for service-to-service
	// @todo ask moddie if this is the place to do this?
	// I've put it here because i want the login service to work the same way, and it needs to message itself
	if serviceToServiceAuth {
		auth.SetCurrentService(Name)
	}

	self.isMultiRegistered = true

	return nil
}

// disconnect our service/endpoints so we can quit cleanly
func (self *discovery) disconnect() error {
	self.Lock()
	defer self.Unlock()

	if self.connected {
		return self.callDiscoveryService("unregister", false)
	}

	return nil
}

// callDiscoveryService sends off a request to register or unregister to the discovery service
func (self *discovery) callDiscoveryService(action string, successState bool) error {
	log.Infof("[Server] Attempting to %s with the discovery service...", action)

	azName, _ := util.GetAwsAZName()
	regSize := reg.size()
	machineClass := os.Getenv("H2O_MACHINE_CLASS")

	endpoints := make([]*register.MultiRequest_Endpoint, regSize)
	i := 0
	for _, endpoint := range reg.iterate() {
		endpoints[i] = &register.MultiRequest_Endpoint{
			Name:      proto.String(endpoint.Name),
			Mean:      proto.Int32(endpoint.Mean),
			Upper95:   proto.Int32(endpoint.Upper95),
			Subscribe: proto.String(endpoint.Subscribe),
		}

		i++
	}

	service := &dscShared.Service{
		Name:        proto.String(Name),
		Description: proto.String(Description),
		Version:     proto.Uint64(Version),
		Source:      proto.String(Source),
		OwnerEmail:  proto.String(OwnerEmail),
		OwnerMobile: proto.String(OwnerMobile),
		OwnerTeam:   proto.String(OwnerTeam),
	}

	request, err := ScopedRequest(
		"com.HailoOSS.kernel.discovery",
		action,
		&register.MultiRequest{
			InstanceId:   proto.String(InstanceID),
			Hostname:     proto.String(self.hostname),
			MachineClass: proto.String(machineClass),
			AzName:       proto.String(azName),
			Service:      service,
			Endpoints:    endpoints,
		},
	)

	if err != nil {
		log.Warnf("[Server] Failed to build request when %sing services", action)
		return err
	}

	// explicitly define timeout, since we're happy to wait
	clientOptions := client.Options{"retries": 0, "timeout": 5 * time.Second}

	rsp := &register.Response{}
	if err := client.Req(request, rsp, clientOptions); err != nil {
		log.Warnf("[Server] Failed to %s services: %v", action, err)
		return err
	}

	// ok -- all done!
	self.connected = successState
	log.Infof("[Server] Successfully %sed with the hive mind!", action)

	return nil
}

func (self *discovery) IsConnected() bool {
	self.RLock()
	defer self.RUnlock()

	return self.connected
}
