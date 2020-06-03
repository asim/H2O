package hintegrate

import (
	"encoding/json"

	"github.com/hailo-platform/H2O/hintegrate/request"
	"github.com/hailo-platform/H2O/hintegrate/validators"
	"github.com/hailo-platform/H2O/hintegrate/variables"
)

// GrantUser will attempt to call `grantuser` on the login service
func GrantUser(c *request.Context, application, uid string, roles []string) (*request.ApiReturn, error) {
	requestData := map[string]interface{}{
		"application": application,
		"uid":         uid,
		"roles":       roles,
	}
	reqJson, _ := json.Marshal(requestData)

	endpoint := "grantuser"
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
