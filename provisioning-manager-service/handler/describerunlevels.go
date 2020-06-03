package handler

import (
	"fmt"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/protobuf/proto"
	descproto "github.com/hailo-platform/H2O/provisioning-manager-service/proto/describerunlevels"
	"github.com/hailo-platform/H2O/provisioning-manager-service/runlevels"
)

func DescribeRunLevels(req *server.Request) (proto.Message, errors.Error) {
	request := &descproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.hailocab.provisioning-manager.describerunlevels", fmt.Sprintf("%v", err))
	}

	return runlevels.DescribeProto(), nil
}
