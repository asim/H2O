package hintegrate

import (
	"encoding/json"
	"net/url"

	"github.com/HailoOSS/hintegrate/request"
	"github.com/HailoOSS/hintegrate/validators"
	"github.com/HailoOSS/hintegrate/variables"
)

type SetRunLevelRequest struct {
	Region string
	Level  int
}

const (
	HALT = iota
	PLATFORM
	CRITICAL
	ESSENTIAL
	DEGRADED
	ALL
)

const (
	// use the gateway service to make this call
	serviceName = "com.HailoOSS.service.gateway"
)

// SetRunLevel calls the setrunlevel endpoint in the provisioning service
func SetRunLevel(c *request.Context, req *SetRunLevelRequest, val ...request.CustomValidationFunc) (*request.ApiReturn, error) {

	payload := map[string]interface{}{
		"region": req.Region,
		"level":  req.Level,
	}

	reqData, _ := json.Marshal(payload)

	endpoint := "setrunlevel"
	postData := map[string]string{
		"service":  serviceName,
		"endpoint": endpoint,
		"request":  string(reqData),
	}

	rsp, err := c.Post().SetHost("callapi_host").
		PostDataMap(postData).SetPath("/v2/h2/call?session_id="+url.QueryEscape(c.Vars.GetVar("admin_token"))).
		Run(serviceName+"."+endpoint, validators.DefaultValCheck(val, validators.Status2xxValidator()))

	vars := variables.NewVariables()

	return &request.ApiReturn{Raw: rsp, ParsedVars: vars}, err
}
