package hintegrate

import (
	"encoding/json"

	"github.com/HailoOSS/hintegrate/request"
	"github.com/HailoOSS/hintegrate/validators"
	"github.com/HailoOSS/hintegrate/variables"
)

type readSessionResponse struct {
	SessionId string `json:"sessId,omitempty"`
	Token     string `json:"token,omitempty"`
}

// ReadSession will attempt to call `readsession` on the login service
func ReadSession(c *request.Context, sessionId string, noRenew bool) (*request.ApiReturn, error) {
	requestData := map[string]interface{}{
		"sessId":  sessionId,
		"noRenew": noRenew,
	}
	reqJson, _ := json.Marshal(requestData)

	endpoint := "readsession"
	postData := map[string]string{
		"service":  serviceName,
		"endpoint": endpoint,
		"request":  string(reqJson),
	}

	rsp, err := c.Post().SetHost("callapi_host").
		PostDataMap(postData).SetPath("/rpc").
		Run(serviceName+"."+endpoint, validators.Status2xxValidator())

	vars := variables.NewVariables()
	parsed := &readSessionResponse{}
	json.Unmarshal(rsp.Body, parsed)

	vars.SetVar("Token", parsed.Token)
	vars.SetVar("SessionId", parsed.SessionId)

	ret := &request.ApiReturn{Raw: rsp, ParsedVars: vars}

	return ret, err
}
