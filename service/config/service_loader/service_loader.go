package service_loader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"

	log "github.com/cihub/seelog"
	nsqlib "github.com/HailoOSS/go-nsq"

	"github.com/HailoOSS/service/config"
	"github.com/HailoOSS/service/nsq"
)

func Init(service string) {
	addr := os.Getenv("H2_CONFIG_SERVICE_ADDR")
	region := os.Getenv("EC2_REGION")
	env := os.Getenv("H2O_ENVIRONMENT_NAME")

	if len(addr) == 0 {
		log.Critical("[Config] Config service address not set")
		log.Flush()
		os.Exit(1)
	}

	loader, err := NewServiceLoader(config.DefaultInstance, addr, service, region, env)
	if err != nil {
		log.Critical("[Config] Failed to created loader: %s", err)
		log.Flush()
		os.Exit(1)
	}

	config.SetLoader(loader)
}

// NewServiceLoader returns a loader that reads config from config service
func NewServiceLoader(c *config.Config, addr, service, region, env string) (*config.Loader, error) {
	// define our hierarchy:
	// H2:BASE
	// H2:BASE:<service-name>
	// H2:REGION:<aws region>
	// H2:REGION:<aws region>:<service-name>
	// H2:ENV:<env>
	// H2:ENV:<env>:<service-name>

	hierarchy := []string{
		"H2:BASE",
		fmt.Sprintf("H2:BASE:%s", service),
		fmt.Sprintf("H2:REGION:%s", region),
		fmt.Sprintf("H2:REGION:%s:%s", region, service),
		fmt.Sprintf("H2:ENV:%s", env),
		fmt.Sprintf("H2:ENV:%s:%s", env, service),
	}

	// Attach service info to Config, this is optional but is used when
	// decrypting secrets
	c.Service = service
	c.Region = region
	c.Env = env

	// construct URL
	if !strings.Contains(addr, "://") {
		addr = "https://" + addr
	}
	addr = strings.TrimRight(addr, "/") + "/compile"
	u, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse config service address: %v", err)
	}
	q := u.Query()
	q.Set("ids", strings.Join(hierarchy, ","))
	u.RawQuery = q.Encode()

	configUrl := u.String()

	log.Infof("[Config] Initialising service loader for service '%s' in region '%s' in '%s' environment via URL %s", service, region, env, configUrl)

	rdr := func() (io.ReadCloser, error) {
		rsp, err := http.Get(configUrl)
		if err != nil {
			log.Errorf("[Config] Failed to load config via %s: %v", configUrl, err)
			return nil, fmt.Errorf("Failed to load config via %s: %v", configUrl, err)
		}
		defer rsp.Body.Close()
		if rsp.StatusCode != 200 {
			log.Errorf("[Config] Failed to load config via %s - status code %v", configUrl, rsp.StatusCode)
			return nil, fmt.Errorf("Failed to load config via %s - status code %v", configUrl, rsp.StatusCode)
		}
		b, _ := ioutil.ReadAll(rsp.Body)

		loaded := make(map[string]interface{})
		err = json.Unmarshal(b, &loaded)
		if err != nil {
			log.Errorf("[Config] Unable to unmarshal loaded config: %v", err)
			return nil, fmt.Errorf("Unable to unmarshal loaded config: %v", err)
		}

		b, err = json.Marshal(loaded["config"])
		if err != nil {
			log.Errorf("[Config] Unable to unmarshal loaded config: %v", err)
			return nil, fmt.Errorf("Unable to unmarshal loaded config: %v", err)
		}
		rdr := ioutil.NopCloser(bytes.NewReader(b))
		return rdr, nil
	}

	changesChan := make(chan bool)
	l := config.NewLoader(c, changesChan, rdr)

	go func() {
		// wait until loaded
		l.Reload()

		// look out for config changes PUBbed via NSQ -- subscribe via a random ephemeral channel
		topic := "config.reload"
		channel := fmt.Sprintf("g%v#ephemeral", rand.Uint32())
		// We expect to receive messages from the federated topic as well
		subscriber, err := nsq.NewDefaultGlobalSubscriber(topic, channel)
		if err != nil {
			log.Warnf("[Config] Failed to create NSQ reader to pickup config changes (fast reload disabled): ch=%v %v", channel, err)
			return
		}

		subscriber.AddHandler(nsqlib.HandlerFunc(func(m *nsqlib.Message) error {
			changesChan <- true
			return nil
		}))

		log.Infof("[Config] Subscribing to config.reload (for fast config reloads)")
		if err := subscriber.Connect(); err != nil {
			log.Warnf("[Config] Failed to connect to NSQ for config changes (fast reload disabled): %v", err)
			return
		}

		// Wait for the Loader to be killed, and then stop the NSQ reader
		l.Wait()
		subscriber.Disconnect()
	}()

	return l, nil
}
