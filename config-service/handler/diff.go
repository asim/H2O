package handler

import (
	"fmt"

	"encoding/json"

	dmp "github.com/hailo-platform/H2O/go-diff/diffmatchpatch"
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/config-service/domain"
	diff "github.com/hailo-platform/H2O/config-service/proto/diff"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

var (
	differ = dmp.New()
)

func pretty(text interface{}) (string, error) {
	var v json.RawMessage

	switch text.(type) {
	case string:
		v = json.RawMessage(text.(string))
	case []byte:
		v = json.RawMessage(text.([]byte))
	default:
		return "", fmt.Errorf("Invalid config type. Must be string or []byte.")
	}

	if len(v) == 0 {
		return "", nil
	}

	var m map[string]interface{}
	err := json.Unmarshal(v, &m)
	if err != nil {
		return "", err
	}

	b, err := json.MarshalIndent(m, "", "    ")
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// Diff will provide a GNU style diff for a configuration at this level in the path with the supplied
// config (for the given ID).
func Diff(req *server.Request) (proto.Message, errors.Error) {
	request := &diff.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailocab.service.config.diff", fmt.Sprintf("%v", err))
	}

	if len(request.GetConfig()) == 0 {
		return nil, errors.BadRequest("com.hailocab.service.config.diff", "Config cannot be blank")
	}

	config, _, err := domain.ReadConfig(request.GetId(), request.GetPath())
	if err != nil && err != domain.ErrPathNotFound {
		return nil, errors.InternalServerError("com.hailocab.service.config.diff", fmt.Sprintf("%v", err))
	}

	p1, err := pretty(config)
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.service.config.diff", fmt.Sprintf("Error parsing existing config: %v", err))
	}

	p2, err := pretty(request.GetConfig())
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.service.config.diff", fmt.Sprintf("Error parsing new config: %v", err))
	}

	deef := differ.DiffMain(p1, p2, true)
	deef = differ.DiffCleanupSemantic(deef)
	patch := differ.PatchToText(differ.PatchMake(deef))

	mdiff, err := json.Marshal(deef)
	if err != nil {
		return nil, errors.InternalServerError("com.hailocab.service.config.diff", fmt.Sprintf("Failed to create response: %v", err))
	}

	rsp := &diff.Response{
		Diff:           proto.String(string(mdiff)),
		Patch:          proto.String(patch),
		ExistingConfig: proto.String(string(config)),
	}

	return rsp, nil
}
