package handler

import (
	"encoding/json"

	"net/http"

	"github.com/hailo-platform/H2O/platform/errors"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func isError(rsp *http.Response, body []byte) errors.Error {
	if rsp.StatusCode != 200 {
		// map to correct error
		var rspBody errorResponse
		if err := json.Unmarshal(body, &rspBody); err != nil {
			return errors.InternalServerError("com.hailocab.service.login.malformedresponse", err.Error())
		}
		if rsp.StatusCode == 400 {
			return errors.BadRequest(rspBody.Code, rspBody.Message)
		} else {
			return errors.InternalServerError(rspBody.Code, rspBody.Message)
		}
	}

	return nil
}
