package request

import (
	"bytes"
	"fmt"
	"github.com/HailoOSS/platform/client"
	"github.com/HailoOSS/platform/raven"
	"github.com/HailoOSS/hshell/integrationtest/result"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type CustomRequstFunc func(i interface{}, b []byte) ([]byte, error)
type CustomValidationFunc func(b []byte) bool

var rabbitOnce sync.Once

func (c CustomRequstFunc) DoRequest(name string, i interface{}, b []byte, val CustomValidationFunc) ([]byte, error) {
	rr := &result.RequestResult{}
	rr.Name = name
	rr.Request = i
	rr.RequestPayload = string(b)
	t := time.Now()
	resp, err := c(i, b)
	dur := time.Since(t)
	rr.Dur = dur
	if err != nil || !val(resp) {
		rr.Success = false
		err = fmt.Errorf("Error: %e :: %s", err, string(resp))
		rr.Response = fmt.Sprintf("%s", err)
	} else {
		rr.Success = true
		rr.Response = string(resp)
	}

	result.Rchan <- rr
	return resp, err
}

func DoHttpRequest(name string, host string, path string, method string, m map[string]string, val CustomValidationFunc) ([]byte, error) {
	hi := &HttpReq{Host: host, Path: path, Method: method}
	postData := ""
	for name, val := range m {
		if len(postData) != 0 {
			postData += "&"
		}
		name = urlEncode(name)
		val = urlEncode(val)
		postData = postData + name + "=" + val
	}
	return CustomRequstFunc(httpRequest).DoRequest(name, hi, []byte(postData), val)
}

func DoRabbitRequest(name string, service string, endpoint string, b []byte, val CustomValidationFunc) ([]byte, error) {
	connectRabbit := func() {
		con := false
		for !con {
			fmt.Println("Connecting Rabbit")
			raven.AmqpUri = fmt.Sprintf("amqp://hailo:hailo@%v:%v", "10.2.2.50", 5672)
			con = <-raven.Connect()
		}
		fmt.Println("connected")
	}

	rabbitOnce.Do(connectRabbit)
	ri := &RabbitReq{Service: service, Endpoint: endpoint}
	return CustomRequstFunc(rabbitRequest).DoRequest(name, ri, b, val)
}

type RabbitReq struct {
	Service  string
	Endpoint string
}

type HttpReq struct {
	Host   string
	Path   string
	Method string
}

func rabbitRequest(i interface{}, b []byte) ([]byte, error) {
	rabreq, ok := i.(*RabbitReq)
	if !ok {
		return nil, fmt.Errorf("Invalid rabbit request parameters: %+v", i)
	}
	var req *client.Request
	req, err := client.NewJsonRequest(rabreq.Service, rabreq.Endpoint, b)
	if err != nil {
		return nil, err
	}
	rsp, err := client.CustomReq(req)
	if err != nil {
		return nil, err
	}
	return rsp.Body(), nil
}

func httpRequest(i interface{}, b []byte) ([]byte, error) {
	httpreqheaders, ok := i.(*HttpReq)
	if !ok {
		return nil, fmt.Errorf("Invalid http request parameters: %+v", i)
	}

	req, err := http.NewRequest(httpreqheaders.Method,
		fmt.Sprintf("%s/%s", httpreqheaders.Host, httpreqheaders.Path),
		bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	buf, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	return buf, nil

}

func urlEncode(s string) string {

	/*pde, err := url.QueryUnescape(s)
	if err == nil {
		s = url.QueryEscape(pde)
	} else {*/
	s = url.QueryEscape(s)
	//}
	return s
}
