package handler

import (
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/dao"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// ReindexUsers is temporary and kicks of user re-indexing for our TS-created index
func ReindexUsers(req *server.Request) (proto.Message, errors.Error) {
	go dao.ReindexUsers()
	return nil, nil
}
