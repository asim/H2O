package handler

import (
	"strconv"
	"time"

	"github.com/hailo-platform/H2O/config-service/domain"
	common "github.com/hailo-platform/H2O/config-service/proto"
	"github.com/hailo-platform/H2O/protobuf/proto"
)

func changeToProto(c *domain.ChangeSet) *common.ChangeMeta {
	return &common.ChangeMeta{
		ChangeId:      proto.String(c.ChangeId),
		Timestamp:     proto.Int64(c.Timestamp.Unix()),
		AuthMechanism: proto.String(c.UserMech),
		UserId:        proto.String(c.UserId),
		Message:       proto.String(c.Message),
	}
}

func changeToFullProto(c *domain.ChangeSet) *common.Change {
	return &common.Change{
		ChangeId:      proto.String(c.ChangeId),
		Id:            proto.String(c.Id),
		Timestamp:     proto.Int64(c.Timestamp.Unix()),
		AuthMechanism: proto.String(c.UserMech),
		UserId:        proto.String(c.UserId),
		Message:       proto.String(c.Message),
		Config:        proto.String(string(c.Body)),
		Path:          proto.String(c.Path),
		OldConfig:     proto.String(string(c.OldConfig)),
	}
}

func changesToFullProto(cs []*domain.ChangeSet) []*common.Change {
	ret := make([]*common.Change, len(cs))
	for i, c := range cs {
		ret[i] = changeToFullProto(c)
	}
	return ret
}

func protoToTime(t *int64, def time.Time) time.Time {
	if t == nil {
		return def
	}
	return time.Unix(*t, 0)
}

func changeToNSQ(action, changeId, id, path, mech, user, message, config, previousConfig string) *NSQEvent {

	return &NSQEvent{
		Id:        changeId,
		Timestamp: strconv.Itoa(int(time.Now().Unix())),
		Type:      "com.hailo-platform/H2O.service.config.event",
		Details: map[string]string{
			"Action":         action,
			"Id":             id,
			"Path":           path,
			"Mech":           mech,
			"User":           user,
			"Message":        message,
			"Config":         config,
			"PreviousConfig": previousConfig,
		},
	}
}
