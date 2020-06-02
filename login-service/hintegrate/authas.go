package hintegrate

import (
	"encoding/json"

	"github.com/HailoOSS/hintegrate/request"
	"github.com/HailoOSS/hintegrate/validators"
	"github.com/HailoOSS/hintegrate/variables"
)

// AuthAs will attempt to call `authas` on the login service
func AuthAs(c *request.Context, application, username, deviceType string) (*request.ApiReturn, error) {
	requestData := map[string]interface{}{
		"application": application,
		"username":    username,
		"deviceType":  deviceType,
	}
	reqJson, _ := json.Marshal(requestData)

	endpoint := "authas"
	postData := map[string]string{
		"service":  serviceName,
		"endpoint": endpoint,
		"request":  string(reqJson),
	}

	rsp, err := c.Post().SetHost("callapi_host").
		PostDataMap(postData).SetPath("/rpc").
		Run(serviceName+"."+endpoint, validators.Status2xxValidator())

	ret := &request.ApiReturn{Raw: rsp, ParsedVars: variables.NewVariables()}

	return ret, err
}
