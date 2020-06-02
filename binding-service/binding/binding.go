package binding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/HailoOSS/binding-service/domain"
	"github.com/HailoOSS/binding-service/util"
	"github.com/HailoOSS/platform/raven"
	plutil "github.com/HailoOSS/platform/util"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	LocalHost         = raven.HOSTNAME
	DefaultRabbitPort = strconv.Itoa(*raven.ADMINPORT)
	isRbFailedOver    bool
	thisAz            string
	thisHttpClient    = &http.Client{}
)

// rabbitmq urls
const (
	BINDING_URL               = "bindings/%%2f/e/%s/%s/%s"
	DEL_BINDING_URL           = "bindings/%%2f/e/%s/%s/%s/%s"
	FED_UPSTREAM_URL          = "parameters/federation-upstream/%%2f/%s"
	POLICIES_URL              = "policies/%%2f/%s"
	EXCHANGE_URL              = "exchanges/%%2f/%s"
	BINDINGS_FOR_EXCHANGE_URL = "exchanges/%%2f/%s/bindings/source"
	EXCHANGES_URL             = "exchanges/%2f"
	QUEUE_URL                 = "queues/%%2f/%s"
)

func getHttpClient() *http.Client {
	return thisHttpClient
}

func Init() {
	var err error
	thisAz, err = plutil.GetAwsAZName()
	if err != nil {
		log.Errorf("Error retrieving AZ name %+v", err)
		panic(fmt.Errorf("Error retrieving AZ name %+v", err))
	}

	isRbFailedOver = IsRabbitFailedOver(getHttpClient(), thisAz)
	log.Debugf("Running in rabbit failover? %v", isRbFailedOver)
	rebindAll(getHttpClient())
	go func() {
		for {
			completed := make(chan bool)
			go func() {
				rebindAll(getHttpClient())
				completed <- true
			}()

			select {
			case <-time.After(REBIND_TIMEOUT):
				panic(fmt.Sprintf("Failed to complete rebinding within timeout of %v, bailing completely to setup fresh.", REBIND_TIMEOUT))
			case <-completed:
			}

			// wait a bit
			time.Sleep(addJitterTo(REBIND_INTERVAL))
		}
	}()

}

func addJitterTo(d time.Duration) time.Duration {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return time.Duration(r.Float64()*float64(d)) + d
}

func CreateBinding(httpClient *http.Client, hostport string, b *domain.BindingDef) (err error) {
	resp, err := createAndSendRequest(httpClient, makeRabbitURL(fmt.Sprintf(BINDING_URL, b.Source, b.GetDestTypeCode(), b.Destination), hostport), "POST", b)
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}

func makeRabbitURL(urlstr string, hostport string) (urlObj *url.URL) {
	return &url.URL{Scheme: "http", Host: hostport, Opaque: "//" + getRabbitCredentials() + hostport + "/api/" + urlstr}
}

func CreateUpstreams(hostnamesArr []string, httpClient *http.Client, hostname string, port int) (err error) {
	// create upstreams
	hostport := hostname + ":" + strconv.Itoa(port)
	for _, hn := range hostnamesArr {
		value := map[string]interface{}{"ack-mode": "no-ack", "expires": 360000, "uri": "amqp://" + hn}
		params := map[string]interface{}{"value": value}
		resp, err := createAndSendRequest(httpClient, makeRabbitURL(fmt.Sprintf(FED_UPSTREAM_URL, hn), hostport), "PUT", params)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return err
		}

	}
	return err
}

func CreateRabbitPolicy(pattern string, name string, httpClient *http.Client, hostname string, port int) (err error) {
	hostport := hostname + ":" + strconv.Itoa(port)
	definition := map[string]interface{}{"federation-upstream-set": "all"}
	params := map[string]interface{}{"pattern": pattern, "definition": definition}
	resp, err := createAndSendRequest(httpClient, makeRabbitURL(fmt.Sprintf(POLICIES_URL, name), hostport), "PUT", params)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		log.Debugf("Created rabbit policy pattern %s name %s", pattern, name)
	}

	return
}

func CreateExchange(exchange *domain.RabbitExchange, httpClient *http.Client) (err error) {
	params := map[string]interface{}{"type": exchange.Xtype, "durable": true}
	if exchange.Options != nil {
		args := make(map[string]interface{})

		for k, v := range exchange.Options {
			args[k] = v
		}
		params["arguments"] = args
	}
	hostport := exchange.Hostname + ":" + strconv.Itoa(exchange.Hostport)
	putRsp, err := createAndSendRequest(httpClient, makeRabbitURL(fmt.Sprintf(EXCHANGE_URL, exchange.Name), hostport), "PUT", params)
	if putRsp != nil {
		defer putRsp.Body.Close()
	}
	if err != nil {
		log.Error("Failed to create exchange ", err)
		return err
	}
	log.Debugf("Created exchange %s", exchange)
	return nil
}

// Get a list of hostnames, one per cluster, useful for hitting services
func getRabbitClusterHosts() ([]domain.RabbitHost, error) {
	resMap, err := util.GetRabbitHosts()
	if err != nil {
		return nil, err
	}
	res := make([]domain.RabbitHost, 0)

	for az, hosts := range resMap {
		res = append(res, domain.RabbitHost{Host: hosts[rand.Int31n(int32(len(hosts)))], AzName: az})
	}
	log.Debugf("Retrieved hostnames %v", res)
	return res, nil
}

func checkError(rsp *http.Response, err error, errString string) (errRes error) {
	errRes = err
	isSuccess := rsp != nil && rsp.StatusCode < 300 && rsp.StatusCode >= 200
	if rsp != nil && !isSuccess && errRes == nil {
		errMsg := errString + " Status code is not successful " + strconv.Itoa(rsp.StatusCode)
		errRes = errors.New(errMsg)
		log.Error(errMsg)
	}
	return errRes
}

// Must close the response body when finished with it
func createAndSendRequest(httpClient *http.Client, url *url.URL, method string, params interface{}) (*http.Response, error) {
	putReq, err := createRequest(url, method, params)
	if err != nil {
		return nil, err
	}
	log.Debugf("Sending request to url %s params %s", url, params)

	rsp, err := httpClient.Do(putReq)
	err = checkError(rsp, err, "Error sending request.")
	return rsp, err
}

func createRequest(u *url.URL, method string, params interface{}) (*http.Request, error) {
	var rdr io.Reader = nil
	if params != nil {
		body, _ := json.Marshal(params)
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, "http:"+u.Opaque, rdr)
	if err != nil {
		log.Debug("Failed to create new request object ", err)
		return nil, err
	}
	req.URL = u // hack to for %2f
	req.SetBasicAuth(raven.USERNAME, raven.PASSWORD)
	if params != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func getRabbitCredentials() (userpass string) {
	return ""
}

func GetAllExchangeBindings(httpClient *http.Client, hostport string, exchange string) ([]*domain.BindingDef, error) {
	return GetAllBindings(httpClient, hostport, raven.EXCHANGE, exchange, domain.EXCHANGE_S)
}

func GetAllQueueBindings(httpClient *http.Client, hostport string, queue string) ([]*domain.BindingDef, error) {
	return GetAllBindings(httpClient, hostport, raven.EXCHANGE, queue, domain.QUEUE_S)
}

func GetAllBindings(httpClient *http.Client, hostport string, fromExchange string, to string, toType domain.DestinationTypeS) ([]*domain.BindingDef, error) {

	// GET Request URL:http://protobroker03-global01-test.i.HailoOSS.com:15672/api/bindings/%2f/e/h2o/e/eu-west-1a/
	resp, err := createAndSendRequest(httpClient, makeRabbitURL(fmt.Sprintf(BINDING_URL, fromExchange, string(toType), to), hostport), "GET", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Debugf("Error reading response, %+v", err)
		return nil, err
	}
	var res []*domain.BindingDef
	log.Debugf("Bindings json %s", string(body))
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Error("Error unmarshalling ", err)
		return nil, err
	}
	return res, nil

}

func GetBindingsForExchange(httpClient *http.Client, hostport string, exchange string) (*[]domain.BindingDef, error) {

	// GET Request URL:http://protobroker03-global01-test.i.HailoOSS.com:15672/api/exchanges/%2f/h2o/bindings/source
	resp, err := createAndSendRequest(httpClient, makeRabbitURL(fmt.Sprintf(BINDINGS_FOR_EXCHANGE_URL, exchange), hostport), "GET", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Debugf("Error reading response, %+v", err)
		return nil, err
	}
	var res []domain.BindingDef
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Error("Error unmarshalling ", err)
		return nil, err
	}
	return &res, nil

}

// Use to delete bindings on remote brokers which point to this service
func DeleteRemoteServiceBindings(httpClient *http.Client, hostport string, service string, thisAz string) error {
	log.Debugf("Retrieving bindings from host %s exchange %s", hostport, thisAz)
	bindings, err := GetAllExchangeBindings(httpClient, hostport, thisAz)
	if err != nil {
		log.Error("Failed to find bindings ", err)
		return err
	}
	for _, val := range bindings {
		args := val.Arguments
		if service == args["service"] {
			DeleteBinding(httpClient, hostport, val)
		}
	}
	return nil
}

// Use to delete bindings on this broker which point from h2o to this service
func DeleteLocalServiceBindings(httpClient *http.Client, hostport string, instanceId string, thisAz string) error {
	log.Debugf("Retrieving bindings from host %s exchange %s", hostport, raven.EXCHANGE)
	bindings, err := GetAllExchangeBindings(httpClient, hostport, raven.EXCHANGE)
	if err != nil {
		log.Error("Failed to find bindings ", err)
		return err
	}
	for _, val := range bindings {
		DeleteBinding(httpClient, hostport, val)
	}
	return nil
}

// Use to delete h2o -> service binding
func DeleteServiceBindings(httpClient *http.Client, hostport string, instanceId string) error {
	log.Debugf("Retrieving bindings from host %s queue %s", hostport, instanceId)
	bindings, err := GetAllQueueBindings(httpClient, hostport, instanceId)
	if err != nil {
		log.Error("Failed to find bindings ", err)
		return err
	}
	for _, val := range bindings {
		// ignore errors because queue is most likely gone anyway
		DeleteBinding(httpClient, hostport, val)

	}
	return nil
}

func DeleteBinding(httpClient *http.Client, hostport string, b *domain.BindingDef) error {
	// DELETE Request URL:http://protobroker01-global01-test.i.HailoOSS.com:15672/api/bindings/%2F/e/h2o/e/eu-west-1c/~_FUDj6QombDT58zwoCtUyA
	// {"vhost":"/","source":"h2o","destination":"eu-west-1c","destination_type":"e","properties_key":"~_FUDj6QombDT58zwoCtUyA"}
	log.Debugf("Deleting binding from %+v", b)

	resp, err := createAndSendRequest(httpClient, makeRabbitURL(fmt.Sprintf(DEL_BINDING_URL, b.Source, string(b.GetDestTypeCode()), b.Destination, b.PropertiesKey), hostport), "DELETE", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Error("Failed to delete binding ", err)
		return err
	}
	return nil
}

func CreateTopicBindingE2Q(httpClient *http.Client, hostport string, from string, destQueue string, topic string) (err error) {
	b := &domain.BindingDef{Source: from, Vhost: "/", Destination: destQueue, DestinationType: string(domain.QUEUE), RoutingKey: topic, Arguments: nil}
	return CreateBinding(httpClient, hostport, b)
}

func GetAllExchanges(httpClient *http.Client, hostport string) (*[]domain.ExchangeDef, error) {
	// GET Request URL:http://protobroker03-global01-test.i.HailoOSS.com:15672/api/exchanges/%2f
	resp, err := createAndSendRequest(httpClient, makeRabbitURL(EXCHANGES_URL, hostport), "GET", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Debugf("Error reading response, %+v", err)
		return nil, err
	}
	var res []domain.ExchangeDef
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Error("Error unmarshalling ", err)
		return nil, err
	}
	return &res, nil

}

// Get names of all remote exchanges (e.g. ones with AZ as name)
func GetAllRemoteExchanges(httpClient *http.Client, hostport string) ([]string, error) {
	mappings, err := GetAllExchanges(httpClient, hostport)
	if err != nil {
		return nil, err
	}
	res := make([]string, 0)
	for _, m := range *mappings {
		name := m.Name
		if name == "" || strings.HasPrefix(name, "amq") || strings.HasPrefix(name, "h2o") || strings.HasPrefix(name, "federation") {
			continue
		}
		res = append(res, name)
	}
	return res, nil
}

func CreateQueue(queue *domain.RabbitQueue, httpClient *http.Client) (err error) {
	params := map[string]interface{}{"durable": true}
	if queue.Options != nil {
		args := make(map[string]interface{})

		for k, v := range queue.Options {
			args[k] = v
		}
		params["arguments"] = args
	}
	hostport := queue.Hostname + ":" + strconv.Itoa(queue.Hostport)
	putRsp, err := createAndSendRequest(httpClient, makeRabbitURL(fmt.Sprintf(QUEUE_URL, queue.Name), hostport), "PUT", params)
	if putRsp != nil {
		defer putRsp.Body.Close()
	}
	if err != nil {
		log.Errorf("Failed to create queue %v", err)
		return err
	}
	log.Debugf("Created queue %s", queue)
	return nil
}

func IsRabbitFailedOver(httpClient *http.Client, thisAz string) bool {
	// need to check the rabbit
	bindings, err := GetBindingsForExchange(httpClient, LocalHost+":"+DefaultRabbitPort, thisAz)
	if err != nil {
		// be optimistic
		log.Errorf("Could not determine if we've failed over, assuming we haven't, %+v", err)
		return false
	}
	isFailedOver := true
	for _, bd := range *bindings {
		// if this az exchange is pointed to h2o then we're on the right cluster
		isFailedOver = bd.Destination != raven.EXCHANGE
		if !isFailedOver {
			break
		}

	}
	return isFailedOver

}
