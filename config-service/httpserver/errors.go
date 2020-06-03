package httpserver

import (
	"encoding/json"
	"net/http"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/platform/errors"
)

type errorBody struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Context []string `json:"context"`
}

// writeError creates a standard error response for a platform error
func writeError(rw http.ResponseWriter, err errors.Error) {
	e := &errorBody{
		Code:    err.Code(),
		Message: err.Description(),
		Context: err.Context(),
	}

	b, marshalErr := json.Marshal(e)
	if marshalErr != nil {
		log.Warnf("Error marshaling the error response into JSON: %v", marshalErr)
	}

	rw.WriteHeader(int(err.HttpCode()))
	rw.Write(b)
}
