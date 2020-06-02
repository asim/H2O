package nsq

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/HailoOSS/service/connhealthcheck"
	"github.com/HailoOSS/service/healthcheck"
	"github.com/mreiferson/go-httpclient"
)

const (
	HealthCheckId   = "com.HailoOSS.service.nsq"
	HighWatermarkId = "com.HailoOSS.service.nsq.highwatermark"
	MaxConnCheckId  = "com.HailoOSS.service.nsq.maxconns"
)

var (
	defaultHttpClient = newHttpClient()
	errNotFound       = errors.New("Channel and topic combination not found.")
)

// HealthCheck asserts we can PUB to NSQ
func HealthCheck() healthcheck.Checker {
	return func() (map[string]string, error) {
		err := Publish("healthcheck", []byte("healthcheck"))
		if err != nil {
			return nil, fmt.Errorf("NSQ Publish failed: %v", err)
		}
		return nil, nil
	}
}

// ChannelPaused asserts that the channel is not paused
func ChannelPaused(topic, channel string) healthcheck.Checker {
	return func() (map[string]string, error) {
		ret := make(map[string]string)
		anyFound := false
		anyPaused := false

		// grab the sub hosts -- which identify NSQD
		subHosts := getHosts(4150, "hailo", "service", "nsq", "subHosts")
		for _, addr := range subHosts {
			addr = strings.Replace(addr, "4150", "4151", -1)
			stats, err := getStats(addr, topic, channel)
			if err != nil {
				return ret, err
			}

			paused, err := isPaused(stats, topic, channel)
			if err != nil {
				if err == errNotFound {
					continue
				}
				// fail hard
				return ret, err
			}
			anyFound = true
			if paused {
				anyPaused = true
			}
			ret[addr] = fmt.Sprintf("%t", paused)
		}

		if !anyFound {
			return ret, fmt.Errorf("Checked %v hosts but did not find any matches for topic '%s' and channel '%s'", len(subHosts), topic, channel)
		}
		if anyPaused {
			return ret, fmt.Errorf("NSQ channel '%v' for topic '%v' is paused", channel, topic)
		}
		return ret, nil
	}
}

// HighWatermark asserts that no individual nsqd has greater than N messages for a channel
// Will fail if the channel doesn't exist on at least one NSQ
func HighWatermark(topic, channel string, mark int) healthcheck.Checker {
	return func() (map[string]string, error) {
		ret := make(map[string]string)
		anyFound := false
		maxDepth := 0

		// grab the sub hosts -- which identify NSQD
		subHosts := getHosts(4150, "hailo", "service", "nsq", "subHosts")
		for _, addr := range subHosts {
			addr = strings.Replace(addr, "4150", "4151", -1)
			stats, err := getStats(addr, topic, channel)
			if err != nil {
				return ret, err
			}

			depth, err := getChannelDepth(stats, topic, channel)
			if err != nil {
				if err == errNotFound {
					continue
				}
				// fail hard
				return ret, err
			}
			anyFound = true
			if depth > maxDepth {
				maxDepth = depth
			}
			ret[addr] = fmt.Sprintf("%v", depth)
		}

		if !anyFound {
			return ret, fmt.Errorf("Checked %v hosts but did not find any matches for topic '%s' and channel '%s'", len(subHosts), topic, channel)
		}
		if maxDepth > mark {
			return ret, fmt.Errorf("NSQ high water mark for topic '%v' and channel '%v' exceeds threshold of %v", topic, channel, mark)
		}
		return ret, nil
	}
}

// MaxNsqdConnHealthCheck asserts that the total number of established tcp connections to all nsqd's fall
// below a given max threshold.
func MaxNsqdConnHealthCheck(maxconns int) healthcheck.Checker {
	return func() (map[string]string, error) {
		pubHosts := getHosts(4150, "hailo", "service", "nsq", "pubHosts")
		subHosts := getHosts(4150, "hailo", "service", "nsq", "subHosts")
		return connhealthcheck.MaxTcpConnections(append(pubHosts, subHosts...), maxconns)()
	}
}

func decodeStats(r io.Reader) (*nsqStats, error) {
	stats := &nsqStats{}
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(stats); err != nil {
		return nil, err
	}
	return stats, nil
}

func getStats(addr, topic, channel string) (*nsqStats, error) {

	url := fmt.Sprintf("http://%v/stats?format=json&topic=%v&channel=%v", addr, topic, channel)
	rsp, err := defaultHttpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	return decodeStats(rsp.Body)
}

func isPaused(stats *nsqStats, topic, channel string) (bool, error) {
	if stats.Data == nil {
		return false, errNotFound
	}
	for _, t := range stats.Data.Topics {
		if t.TopicName == topic {
			// find channel
			for _, ch := range t.Channels {
				if ch.ChannelName == channel {
					return ch.Paused, nil
				}
			}
		}
	}

	return false, errNotFound
}

// getChannelDepth calls NSQ via HTTP to get stats and then extracts depth of topic/channel
func getChannelDepth(stats *nsqStats, topic, channel string) (int, error) {
	if stats.Data == nil {
		return 0, errNotFound
	}
	for _, t := range stats.Data.Topics {
		if t.TopicName == topic {
			// find channel
			for _, ch := range t.Channels {
				if ch.ChannelName == channel {
					return ch.Depth, nil
				}
			}
		}
	}

	return 0, errNotFound
}

func newHttpClient() *http.Client {
	transport := &httpclient.Transport{}
	transport.ConnectTimeout = time.Second
	transport.RequestTimeout = time.Second
	transport.ResponseHeaderTimeout = time.Second
	client := &http.Client{Transport: transport}
	return client
}

// representation of stats results:

type nsqStats struct {
	Data *nsqStatsData `json:"data"`
}

type nsqStatsData struct {
	Topics []*nsqStatsTopic `json:"topics"`
}

type nsqStatsTopic struct {
	TopicName string `json:"topic_name"`
	Channels  []*nsqStatsChannel
}

type nsqStatsChannel struct {
	ChannelName string `json:"channel_name"`
	Depth       int    `json:"depth"`
	Paused      bool   `json:"paused"`
}
