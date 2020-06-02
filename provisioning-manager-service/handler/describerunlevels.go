package handler

import (
	"fmt"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
	"github.com/HailoOSS/protobuf/proto"
	descproto "github.com/HailoOSS/provisioning-manager-service/proto/describerunlevels"
	"github.com/HailoOSS/provisioning-manager-service/runlevels"
)

func DescribeRunLevels(req *server.Request) (proto.Message, errors.Error) {
	request := &descproto.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.provisioning-manager.describerunlevels", fmt.Sprintf("%v", err))
	}

	return runlevels.DescribeProto(), nil
}
