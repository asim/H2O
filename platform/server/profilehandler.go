package server

import (
	"fmt"

	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/profile"
	"github.com/HailoOSS/protobuf/proto"

	dcprofile "github.com/jono-macd/profile"

	profilestartproto "github.com/HailoOSS/platform/proto/profilestart"
	profilestopproto "github.com/HailoOSS/platform/proto/profilestop"
)

// profileStartHandler handles inbound requests to `profilestart` endpoint
func profileStartHandler(req *Request) (proto.Message, errors.Error) {
	request := &profilestartproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.kernel.platform.profilestart", fmt.Sprintf("%v", err))
	}

	cfg := &dcprofile.Config{}

	switch request.GetType() {
	case profilestartproto.ProfileType_CPU:
		cfg.CPUProfile = true
	case profilestartproto.ProfileType_MEMORY:
		cfg.MemProfile = true
	case profilestartproto.ProfileType_BLOCK:
		cfg.BlockProfile = true
	}

	output := profile.S3Output
	switch request.GetOutput() {
	case profilestartproto.Output_S3:
		output = profile.S3Output
	case profilestartproto.Output_FILE:
		output = profile.FileOutput
	}

	if !cfg.CPUProfile && !cfg.MemProfile && !cfg.BlockProfile {
		return nil, errors.BadRequest("com.HailoOSS.kernel.platform.profilestart", "Unsupported profile type, please use: cpu, memory or block")
	}

	if err := profile.StartProfiling(cfg, output, fmt.Sprintf("%s-%v", Name, Version), request.GetId()); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.kernel.platform.profilestart", fmt.Sprintf("Unable to start profile %v", err.Error()))
	}

	return &profilestartproto.Response{}, nil
}

// profileStopHandler handles inbound requests to `profilestop` endpoint
func profileStopHandler(req *Request) (proto.Message, errors.Error) {
	id, profileOutput, binaryOutput, err := profile.StopProfiling()
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.kernel.platform.profilestop", fmt.Sprintf("Unable to stop profile %v", err.Error()))
	}
	pr := &profilestopproto.Response{
		Id:            proto.String(id),
		ProfileOutput: proto.String(profileOutput),
		BinaryOutput:  proto.String(binaryOutput),
	}

	return pr, nil
}
