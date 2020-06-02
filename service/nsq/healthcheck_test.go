package nsq

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestIsPaused(t *testing.T) {
	stats, err := decodeStats(strings.NewReader(sampleJson))
	if err != nil {
		t.Fatal(err)
	}
	paused, err := isPaused(stats, "chargeset.version", "charging-service")
	if err != nil {
		t.Fatal(err)
	}
	if !paused {
		t.Fatal("Expected paused to be true, got false")
	}
}

func TestGetChannelDepth(t *testing.T) {
	stats, err := decodeStats(strings.NewReader(sampleJson))
	if err != nil {
		t.Fatal(err)
	}
	depth, err := getChannelDepth(stats, "login.sessionexpire", "expire")
	if err != nil {
		t.Fatal(err)
	}
	if depth != 250 {
		t.Errorf("Expected %d, Got %d", 250, depth)
	}
}

func TestGetStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		io.WriteString(rw, sampleJson)
	}))
	defer server.Close()
	addr, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := getStats(addr.Host, "config.reload", "healthcheck")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Data == nil {
		t.Fatal("Stats data should not be nil")
	}
	topicCount := len(stats.Data.Topics)
	if topicCount != 5 {
		t.Fatalf("Expected 5 topics, got %d", topicCount)
	}
}

// Json with a paused channel and a channel with backed up items
var sampleJson = `{
  "status_code": 200,
  "status_txt": "OK",
  "data": {
    "topics": [
      {
        "topic_name": "chargeset.version",
        "channels": [
          {
            "channel_name": "charging-service",
            "depth": 0,
            "backend_depth": 0,
            "in_flight_count": 0,
            "deferred_count": 0,
            "message_count": 555,
            "requeue_count": 0,
            "timeout_count": 0,
            "clients": [],
            "paused": true
          }
        ],
        "depth": 0,
        "backend_depth": 0,
        "message_count": 664
      },
      {
        "topic_name": "config.reload",
        "channels": [],
        "depth": 0,
        "backend_depth": 0,
        "message_count": 0
      },
      {
        "topic_name": "healthcheck",
        "channels": [],
        "depth": 52,
        "backend_depth": 52,
        "message_count": 0
      },
      {
        "topic_name": "login.sessionexpire",
        "channels": [
          {
            "channel_name": "expire",
            "depth": 250,
            "backend_depth": 0,
            "in_flight_count": 0,
            "deferred_count": 0,
            "message_count": 0,
            "requeue_count": 0,
            "timeout_count": 0,
            "clients": [],
            "paused": false
          }
        ],
        "depth": 0,
        "backend_depth": 0,
        "message_count": 0
      },
      {
        "topic_name": "testchargeset.version",
        "channels": [
          {
            "channel_name": "charging-service",
            "depth": 16,
            "backend_depth": 0,
            "in_flight_count": 0,
            "deferred_count": 0,
            "message_count": 35,
            "requeue_count": 55,
            "timeout_count": 55,
            "clients": [],
            "paused": false
          }
        ],
        "depth": 0,
        "backend_depth": 0,
        "message_count": 36
      }
    ]
  }
}`
