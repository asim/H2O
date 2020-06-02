package handler

import (
	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/dao"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// ReindexUsers is temporary and kicks of user re-indexing for our TS-created index
func ReindexUsers(req *server.Request) (proto.Message, errors.Error) {
	go dao.ReindexUsers()
	return nil, nil
}
