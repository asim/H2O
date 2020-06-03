package handler

import (
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/hailo-platform/H2O/protobuf/proto"

	"github.com/hailo-platform/H2O/login-service/auther"
	"github.com/hailo-platform/H2O/login-service/dao"
	readsession "github.com/hailo-platform/H2O/login-service/proto/readsession"
	"github.com/hailo-platform/H2O/platform/errors"
	"github.com/hailo-platform/H2O/platform/server"
)

// ReadSession will retrieve token/session information based on session ID.
// By default this will auto-renew tokens for active sessions that have expired
// or near-expired tokens - although this can be disabled.
func ReadSession(req *server.Request) (proto.Message, errors.Error) {
	request := &readsession.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.hailo-platform/H2O.service.login.readsession.unmarshal", err.Error())
	}

	log.Debugf("Session read request from %s for session %s", req.From(), request.GetSessId())

	sess, err := dao.ReadSession(request.GetSessId())
	if err != nil {
		return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.readsession.dao", err.Error())
	}
	if sess == nil {
		return nil, errors.NotFound("com.hailo-platform/H2O.service.login.readsession", fmt.Sprintf("Session %v not found", request.GetSessId()))
	}

	log.Debugf("Got back sess %#v", sess)

	// noRenew means we DON'T want to automatically extend the lifetime of this session
	if !request.GetNoRenew() {
		updated, err := auther.AutoRenew(sess)
		if err != nil {
			return nil, errors.InternalServerError("com.hailo-platform/H2O.service.login.readsession.autorenew", err.Error())
		}
		// did we update?
		if updated != nil {
			sess = updated
		}
	}

	rsp := &readsession.Response{
		SessId: proto.String(sess.Id),
		Token:  proto.String(sess.Token.String()),
	}

	return rsp, nil
}
