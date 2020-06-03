package handler

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	api "github.com/hailo-platform/H2O/api-proxy/proto/api"
)

func TestHttpFormEncodedPostReqToProto(t *testing.T) {
	values := make(url.Values)
	values.Set("foo", "bar")
	values.Set("baz", "bing")
	body := values.Encode()

	req, _ := http.NewRequest("POST", "http://localhost", bytes.NewReader([]byte(body)))

	apiReq, err := httpRequestToProto(req)
	if err != nil {
		t.Fatalf("Unexpected HTTP -> proto marshaling error: %v", err)
	}

	assertKVs(t, apiReq.GetPost(), map[string]string{
		"foo": "bar",
		"baz": "bing",
	})
	assertKVs(t, apiReq.GetGet(), map[string]string{})
}

func TestHttpFormEncodedPostReqToProtoWithQuery(t *testing.T) {
	values := make(url.Values)
	values.Set("foo", "bar")
	values.Set("baz", "bing")
	body := values.Encode()

	req, _ := http.NewRequest("POST", "http://localhost?barbar=foobarbazbing", bytes.NewReader([]byte(body)))

	apiReq, err := httpRequestToProto(req)
	if err != nil {
		t.Fatalf("Unexpected HTTP -> proto marshaling error: %v", err)
	}

	assertKVs(t, apiReq.GetPost(), map[string]string{
		"foo": "bar",
		"baz": "bing",
	})
	assertKVs(t, apiReq.GetGet(), map[string]string{
		"barbar": "foobarbazbing",
	})
}

func TestHttpFormEncodedPostReqToProtoWithQueryAndBasicAuth(t *testing.T) {
	values := make(url.Values)
	values.Set("foo", "bar")
	values.Set("baz", "bing")
	body := values.Encode()

	req, _ := http.NewRequest("POST", "http://localhost?barbar=foobarbazbing", bytes.NewReader([]byte(body)))
	req.SetBasicAuth("Aladdin", "open sesame")
	req.Header.Add("Something-Silly", "golang")

	apiReq, err := httpRequestToProto(req)
	if err != nil {
		t.Fatalf("Unexpected HTTP -> proto marshaling error: %v", err)
	}

	assertKVs(t, apiReq.GetPost(), map[string]string{
		"foo": "bar",
		"baz": "bing",
	})
	assertKVs(t, apiReq.GetGet(), map[string]string{
		"barbar": "foobarbazbing",
	})
	assertHeaders(t, apiReq.GetHeader(), []string{
		"Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==",
	})
}

func TestHttpGetIgnoredSessionIdAndApiToken(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://localhost?session_id=foobarbazbing", nil)
	apiReq, err := httpRequestToProto(req)
	if err != nil {
		t.Fatalf("Unexpected HTTP -> proto marshaling error: %v", err)
	}
	assertKVs(t, apiReq.GetGet(), map[string]string{})

	req, _ = http.NewRequest("GET", "http://localhost?api_token=foobarbazbing", nil)
	apiReq, err = httpRequestToProto(req)
	if err != nil {
		t.Fatalf("Unexpected HTTP -> proto marshaling error: %v", err)
	}
	assertKVs(t, apiReq.GetGet(), map[string]string{})
}

func TestHttpJsonPostReqToProto(t *testing.T) {
	json := `{"foo":"bar"}`
	req, _ := http.NewRequest("POST", "http://localhost", bytes.NewReader([]byte(json)))
	req.Header.Set("Content-Type", "application/json")

	apiReq, err := httpRequestToProto(req)
	if err != nil {
		t.Fatalf("Unexpected HTTP -> proto marshaling error: %v", err)
	}

	assertKVs(t, apiReq.GetPost(), map[string]string{})
	assertKVs(t, apiReq.GetGet(), map[string]string{})
	if apiReq.GetBody() != json {
		t.Errorf("Expecting JSON body %s - got %s", json, apiReq.GetBody())
	}
}

func assertKVs(t *testing.T, values []*api.Request_Pair, expected map[string]string) {
	if len(values) != len(expected) {
		t.Errorf("Expecting %v K/V values, got %v", len(expected), len(values))
	}

	for k, v := range expected {
		found := false
		for _, val := range values {
			if val.GetKey() == k && val.GetValue() == v {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Failed to find K=%s, V=%s", k, v)
		}
	}
}

func assertHeaders(t *testing.T, values []string, expected []string) {
	if len(values) != len(expected) {
		t.Errorf("Expecting %v header values, got %v", len(expected), len(values))
	}
	for _, h := range expected {
		found := false
		for _, vh := range values {
			if vh == h {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Failed to find Header=%s", h)
		}
	}
}
