package binding

import (
	"fmt"
	log "github.com/cihub/seelog"
	"net/http"
	"time"

	"github.com/HailoOSS/binding-service/dao"
	"github.com/HailoOSS/binding-service/domain"
	instances "github.com/HailoOSS/discovery-service/proto/instances"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/raven"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/service/sync"
)

var (
	localServices map[string]bool = map[string]bool{"com.HailoOSS.kernel.binding": true} // services which shouldn't cross AZs
)

const (
	REBIND_INTERVAL = 3 * time.Minute
	REBIND_TIMEOUT  = 2 * time.Minute // after this time, we treat rebinding as failed, and bail out compeltely
	LOCK_STRING     = "%s%s"
)

func PostConnectHandler() {
	// register serviceup topic listener
	httpClient := http.Client{}
	subTopic := "com.HailoOSS.kernel.discovery.serviceup"
	err := CreateTopicBindingE2Q(&httpClient, LocalHost+":"+DefaultRabbitPort, raven.TOPIC_EXCHANGE, server.InstanceID, subTopic)
	if err != nil {
		log.Error("Failed to subscribe to ", subTopic, err)
		panic(err)
	}
	log.Debug("Subscribed to ", subTopic)
	subTopic = "com.HailoOSS.kernel.discovery.servicedown"
	err = CreateTopicBindingE2Q(&httpClient, LocalHost+":"+DefaultRabbitPort, raven.TOPIC_EXCHANGE, server.InstanceID, subTopic)
	if err != nil {
		log.Error("Failed to subscribe to ", subTopic, err)
		panic(err)
	}
	log.Debug("Subscribed to ", subTopic)

}

// For rebinding, our responsibility is to make sure the bindings for our local services are correct across all clusters
func rebindAll(httpClient *http.Client) {
	log.Debug("Rebinding all service instances")

	request, err := server.ScopedRequest(
		"com.HailoOSS.kernel.discovery",
		"instances",
		&instances.Request{},
	)
	if err != nil {
		log.Error(err)
		return
	}

	response := &instances.Response{}

	if err := client.Req(request, response); err != nil {
		log.Error(err)
		return
	}

	remoteRunning := make(map[string]*domain.Service)
	inst := response.GetInstances()
	for _, i := range inst {
		s := domain.ServiceFromInstancesProto(i)
		if i.GetAzName() == thisAz {
			// Set up this service instance on this cluster
			if err := SetupService(s); err != nil {
				log.Errorf("Error while attempting to setup service %#v %s", s, err.Description())
			}
		} else {
			remoteRunning[i.GetAzName()+i.GetServiceName()] = s
		}
	}
	log.Debug("Rebinding all service instances complete")
	if isRbFailedOver {
		// make sure we teardown everything that is pointing to the old AZ which is now down
		teardownRemotesForAZ(httpClient, thisAz)
	} else {
		// clean up this cluster
		teardownMissing(thisAz, remoteRunning)
	}
}

// Tear down all bindings which point to this AZ - Use in failover scenario
func teardownRemotesForAZ(httpClient *http.Client, az string) {
	log.Debugf("Tearing down remotes for AZ %s", az)
	hosts, err := getRabbitClusterHosts()
	if err != nil {
		log.Errorf("Error while retrieving hostnames, %+v", err)
		return
	}
	for _, host := range hosts {
		if host.AzName == thisAz {
			continue
		}
		hostPort := host.Host + ":" + DefaultRabbitPort
		bindings, err := GetAllExchangeBindings(httpClient, hostPort, az)
		if err != nil {
			log.Debugf("Error getting all exchange bindings, %+v", err)
			return
		}
		for _, b := range bindings {
			DeleteBinding(httpClient, hostPort, b)
		}

	}
	log.Debugf("Tearing down remotes for AZ %s complete", az)

}

// For tearing down our responsibility is to make sure our bindings in our local cluster are correct
func teardownMissing(thisAz string, remoteRunning map[string]*domain.Service) {
	log.Debug("Tearing down any missing services")
	hostport := LocalHost + ":" + DefaultRabbitPort
	// find all exchanges
	exchNames, err := GetAllRemoteExchanges(getHttpClient(), hostport)
	if err != nil {
		log.Errorf("Error determining remote exchanges %+v", err)
		return
	}
	// for each exchange X
	// - check for bindings h2o -> X
	// - call discovery service instances(X) where X is an azName
	// - match bindings to available instances and delete ones that are unavailable (its ok to be overly aggressive with deleting bindings because bindings will get fixed in next run)
	for _, x := range exchNames {
		if x == thisAz {
			continue
		}
		bindings, err := GetAllExchangeBindings(getHttpClient(), hostport, x)
		if err != nil {
			log.Errorf("Error getting bindings for exchange %s %+v", x, err)
			continue // let's just try to clear up as much as we can so soldier on
		}
		for _, b := range bindings {
			// does it exist?
			if _, ok := remoteRunning[x+b.Arguments["service"].(string)]; !ok {
				// remove
				log.Debug("Deleting binding for missing service %+v", b)
				DeleteBinding(getHttpClient(), hostport, b)
			}
		}
	}

	log.Debug("Tearing down any missing services complete")

}

func SetupService(s *domain.Service) errors.Error {

	if thisAz != s.AzName {
		return nil // not in the corresponding AZ
	}

	log.Debugf("Setting up service %+v", s)

	lock, err := getLock(s.Service, s.AzName)
	defer lock.Unlock()
	if err != nil {
		log.Errorf("Failed to acquire lock to setup up process %+v", err)
		return errors.InternalServerError("com.HailoOSS.kernel.binding.setupservice", err.Error())
	}

	if err != nil {
		log.Errorf("Failed to acquire lock to setup up process %+v", err)
		return errors.InternalServerError("com.HailoOSS.kernel.binding.setupservice", err.Error())
	}
	log.Debug("Acquired lock")

	rules, err := dao.GetRules(s.Service)
	if err != nil {
		log.Errorf("Error retrieving binding rules %+v", err)
		return errors.InternalServerError("com.HailoOSS.kernel.binding.setupservice", err.Error())
	}
	if rules == nil || len(rules) == 0 {
		// sort out a default rule with weight 100. This means that only < 1% of messages will go over the federation links
		rules = []*domain.Rule{&domain.Rule{Service: s.Service, Weight: 100, Version: s.Version}}
	}

	// create new binding before deleting any old ones, that way the queue is always receiving messages
	b := domain.BindingDefFromService(s)
	applyRules(rules, b, s)
	hostport := LocalHost + ":" + DefaultRabbitPort
	err = CreateBinding(getHttpClient(), hostport, b)
	if err != nil {
		return errors.InternalServerError("com.HailoOSS.kernel.binding.setupservice", fmt.Sprintf("Error while creating E2Q binding h2o -> %v. %v", b.Destination, err))
	}

	bindings, err := GetAllQueueBindings(getHttpClient(), hostport, b.Destination)
	if err != nil {
		return errors.InternalServerError("com.HailoOSS.kernel.binding.setupservice", fmt.Sprintf("Error while querying current bindings h2o -> %v. %v", b.Destination, err))
	}
	log.Debugf("There are %d bindings for queue %s", len(bindings), b.Destination)
	if len(bindings) > 1 {
		deleted := 0
		// need to trim the bindings
		for _, currBinding := range bindings {
			if !b.Equals(currBinding) {
				if deleted == len(bindings)-1 {
					// safety net. Shouldn't be needed but saw some weird stuff
					log.Debugf("Binding %+v doesn't equal %+v but won't delete because it will leave us with zero bindings", b, currBinding)
					break
				}
				log.Debugf("Binding %+v doesn't equal %+v", b, currBinding)
				// delete, ignore errors
				DeleteBinding(getHttpClient(), hostport, currBinding)
				deleted++
			}
		}
	}

	for _, sub := range s.Subscriptions {
		if sub != "" {
			err := CreateTopicBindingE2Q(getHttpClient(), LocalHost+":"+DefaultRabbitPort, raven.TOPIC_EXCHANGE, s.Instance, sub)
			if err != nil {
				return errors.InternalServerError("com.HailoOSS.kernel.binding.setupservice", fmt.Sprintf("Error while creating E2Q binding h2o.topic -> %v. %v", s.Instance, err))
			}
		}
	}

	if !localServices[s.Service] {
		if isRbFailedOver {
			// don't do any of the remote stuff
			log.Debug("We've failed over so not doing any remote bindings")
			return nil
		}
		hosts, err := getRabbitClusterHosts()
		if err != nil {
			return errors.InternalServerError("com.HailoOSS.kernel.binding.setupservice", fmt.Sprintf("Error while retrieving hostnames %v", err))
		}

		for _, host := range hosts {
			// Intentionally do not apply binding rules, leave blank so we always create same binding for a service
			// regardless of rules. This reduces the number of bindings created. Also means that it reduces cross AZ traffic
			// if rules are applied since x-weight defaults to 1
			if host.AzName == thisAz {
				continue
			}
			eb := domain.ExchangeBindingDefFromService(s, thisAz)
			remoteHostPort := host.Host + ":" + DefaultRabbitPort
			err = CreateBinding(getHttpClient(), remoteHostPort, eb)
			if err != nil {
				return errors.InternalServerError("com.HailoOSS.kernel.binding.setupservice", fmt.Sprintf("Error while creating E2E binding h2o -> %v on %v. %v", thisAz, host, err))
			}

		}
	}

	return nil
}

func applyRules(rules []*domain.Rule, b *domain.BindingDef, s *domain.Service) {
	if rules != nil {
		for _, r := range rules {
			// TODO what happens if more than one is applicable?
			if r.IsApplicable(s) {
				for k, v := range r.GetRuleMap() {
					b.Arguments[k] = v
				}

			}

		}
	}

}

func TeardownService(service string, queue string, azName string) errors.Error {

	if thisAz != azName {
		return nil // not in the corresponding AZ
	}

	log.Debugf("Tearing down service %s", service)

	// remove binding - normally auto removed since queue should die BUT if service that's down still has it's connection but not responding then we need to remove binding
	DeleteLocalServiceBindings(getHttpClient(), LocalHost+":"+DefaultRabbitPort, raven.EXCHANGE, queue)
	log.Debugf("Tearing down service done %+v", service)
	return TeardownRemoteServiceBindings(getHttpClient(), service, azName)
}

func TeardownRemoteServiceBindings(httpClient *http.Client, service string, azName string) errors.Error {
	if !localServices[service] {

		lock, err := getLock(service, azName)
		defer lock.Unlock()
		if err != nil {
			log.Errorf("Failed to acquire lock to tear down process %+v", err)
			return errors.BadRequest("com.HailoOSS.kernel.binding.teardownservice", err.Error())
		}

		if err != nil {
			log.Errorf("Failed to acquire lock to tear down process %+v", err)
			return errors.BadRequest("com.HailoOSS.kernel.binding.teardownservice", err.Error())
		}
		log.Debug("Acquired lock")
		last, err := isLastInstanceInAz(httpClient, LocalHost+":"+DefaultRabbitPort, service, azName)
		if err != nil {
			log.Error("Error while finding last instance ", err)
		} else if last {

			log.Debug("Last instance in AZ, unbinding in other AZs")
			hosts, err := getRabbitClusterHosts()
			if err != nil {
				return errors.InternalServerError("com.HailoOSS.kernel.binding.teardownservice", fmt.Sprintf("Error while retrieving hostnames %v", err))
			}

			for _, host := range hosts {
				if host.AzName == thisAz {
					continue
				}

				err = DeleteRemoteServiceBindings(httpClient, host.Host+":"+DefaultRabbitPort, service, azName)
				if err != nil {
					return errors.InternalServerError("com.HailoOSS.kernel.binding.setupservice", fmt.Sprintf("Error while deleting service bindings %v", err))
				}
			}
		}
	}
	return nil
}

func getLock(service string, azName string) (sync.Lock, error) {
	lockRt := fmt.Sprintf(LOCK_STRING, service, azName)
	return sync.RegionLock([]byte(lockRt))
}

func isLastInstanceInAz(httpClient *http.Client, hostport string, serviceName string, azName string) (bool, error) {
	// Check all h2o -> q bindings for this service name. If none available then this was last instance
	bindings, err := GetBindingsForExchange(httpClient, hostport, raven.EXCHANGE)
	if err != nil {
		log.Errorf("Can't tell whether this is the last instance in the AZ %+v", err)
		return false, err
	}
	found := false
	log.Debugf("Checking last instance %s", serviceName)
	for _, v := range *bindings {
		args := v.Arguments
		if args != nil {
			found = args["service"] == serviceName && v.DestinationType == "queue"
			if found {
				break
			}
		}
	}
	return !found, nil
}
