package elasticsearch

import (
	"net/http"
	"strconv"

	"github.com/HailoOSS/go-hailo-lib/log"
	"github.com/HailoOSS/service/config"
	"github.com/mattbaird/elastigo/lib"
)

func NewConnection() *elastigo.Conn {
	c := elastigo.NewConn()

	ch := config.SubscribeChanges()
	go func() {
		for _ = range ch {
			loadEndpointConfig(c, nil)
		}
	}()

	loadEndpointConfig(c, nil)

	return c
}

func IndexExists(c *elastigo.Conn, index string) (bool, error) {
	r, err := c.NewRequest("HEAD", "/"+index, "")

	var client = r.Client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(r.Request)

	if err != nil {
		return false, err
	}

	return !(resp.StatusCode == 404), nil
}

func loadEndpointConfig(c *elastigo.Conn, hosts []string) {
	log.Info("[ElasticSearch] Loading config")

	port := config.AtPath("hailo", "service", "elasticsearch", "port").AsInt(9200)

	if hosts == nil {
		hosts = config.AtPath("hailo", "service", "elasticsearch", "hosts").AsHostnameArray(port)
	}

	if len(hosts) == 0 {
		hosts = append(hosts, "localhost:9200")
	}

	// Set these hosts in the Elasticsearch library
	// This will initialise a host pool which uses an Epsilon Greedy algorithm to find healthy hosts
	// and send to requests to them, and not unhealthy or slow hosts
	if port == 443 {
		c.Protocol = "https"
	}
	c.SetPort(strconv.Itoa(port))
	c.SetHosts(hosts)

	log.Infof("[ElasticSearch] hosts loaded: %v", hosts)
}
