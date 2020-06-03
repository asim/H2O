package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hailo-platform/H2O/binding-service/domain"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpErrorCheck(t *testing.T) {
	resp := http.Response{}
	resp.StatusCode = 200
	err := checkError(&resp, nil, "Test error")
	if err != nil {
		t.Error("Error should be nil")
	}
	resp.StatusCode = 201
	err = checkError(&resp, nil, "Test error")
	if err != nil {
		t.Error("Error should be nil")
	}
	resp.StatusCode = 400
	err = checkError(&resp, nil, "Test error")
	if err == nil {
		t.Error("Error should not be nil")
	}
	resp.StatusCode = 200
	err = checkError(&resp, errors.New("Some err"), "Test error")
	if err == nil {
		t.Error("Error should not be nil")
	}
	resp.StatusCode = 500
	err = checkError(&resp, nil, "Test error")
	if err == nil {
		t.Error("Error should not be nil")
	}

}

func TestHttpRequestCreation(t *testing.T) {
	req, _ := createRequest(makeRabbitURL("bindings/%2f/e/h2o/q/1234", "localhost:15672"), "GET", nil)
	if req.Method != "GET" {
		t.Error("Method not set properly", req.Method)
	}
	if req.Host != "localhost:15672" {
		t.Error("Host not set properly", req.Host)
	}
	if req.URL.RequestURI() != "http://localhost:15672/api/bindings/%2f/e/h2o/q/1234" {
		t.Errorf("URI not set properly %+v", req.URL.RequestURI())
	}

	params := make(map[string]interface{})
	params["foobar"] = "baz"
	params["hello"] = "world"
	req, _ = createRequest(makeRabbitURL("bindings/%2f/e/h2o/q/1234", "localhost:15672"), "POST", params)
	if req.Method != "POST" {
		t.Error("Method not set properly", req.Method)
	}
	if req.Host != "localhost:15672" {
		t.Error("Host not set properly", req.Host)
	}
	if req.URL.RequestURI() != "http://localhost:15672/api/bindings/%2f/e/h2o/q/1234" {
		t.Errorf("URI not set properly %+v", req.URL.RequestURI())
	}

	body, _ := ioutil.ReadAll(req.Body)
	var res map[string]interface{}
	json.Unmarshal(body, &res)
	if res["foobar"].(string) != "baz" || res["hello"].(string) != "world" {
		t.Error("Params not marshalled correctly")
	}

	req, _ = createRequest(makeRabbitURL("bindings/%2f/e/h2o/q/1234", "localhost:15672"), "DELETE", nil)
	if req.Method != "DELETE" {
		t.Error("Method not set properly", req.Method)
	}
	if req.Host != "localhost:15672" {
		t.Error("Host not set properly", req.Host)
	}
	if req.URL.RequestURI() != "http://localhost:15672/api/bindings/%2f/e/h2o/q/1234" {
		t.Errorf("URI not set properly %+v", req.URL.RequestURI())
	}
}

func TestCreateBinding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var bindingDef domain.BindingDef
		bytes, _ := ioutil.ReadAll(r.Body)
		err := json.Unmarshal(bytes, &bindingDef)
		if err != nil {
			t.Fail()
		}
		fmt.Printf("Received request %#v\n", r)

		switch {
		case r.URL.Path == "/api/bindings///e/h2o/q/server-com.hailo-platform/H2O.service.foobar-1234567890":
			v, ok := bindingDef.Arguments["service"]
			if !ok {
				t.Error("Missing 'service' argument")

			}
			if v.(string) != "com.hailo-platform/H2O.service.foobar" {
				t.Error("'service' argument incorrect", v)

			}
			v, ok = bindingDef.Arguments["x-match"]
			if !ok {
				t.Error("Missing 'x-match' argument")

			}
			if v.(string) != "all" {
				t.Error("'x-match' argument incorrect", v)
			}
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusBadRequest)
			t.Errorf("Error, incorrect URL called %#v", r.URL)
		}
	}))
	defer srv.Close()
	srvURL := srv.URL[7:] // remove the leading http://
	hc := &http.Client{}
	b := &domain.BindingDef{Source: "h2o",
		Vhost:           "/",
		Destination:     "server-com.hailo-platform/H2O.service.foobar-1234567890",
		DestinationType: string(domain.QUEUE),
		RoutingKey:      "com.hailo-platform/H2O.service.foobar",
		Arguments:       map[string]interface{}{"service": "com.hailo-platform/H2O.service.foobar", "x-match": "all"}}
	err := CreateBinding(hc, srvURL, b)
	if err != nil {
		t.Error("Error creating binding ", err)
	}
}
