package handler

import (
	"fmt"
	"time"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/config-service/domain"
	changelog "github.com/HailoOSS/config-service/proto/changelog"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// ChangeLog will read a time series of changes made within a range
func ChangeLog(req *server.Request) (proto.Message, errors.Error) {
	request := &changelog.Request{}
	if err := req.Unmarshal(request); err != nil {
		return nil, errors.BadRequest("com.HailoOSS.service.config.changelog", fmt.Sprintf("%v", err))
	}

	id := request.GetId()
	start := protoToTime(request.RangeStart, time.Now().Add(-time.Hour))
	end := protoToTime(request.RangeEnd, time.Now())
	count := 10 // @todo allow specify in proto
	lastId := request.GetLastId()

	var chs []*domain.ChangeSet
	var last string
	var err error

	if len(id) == 0 {
		chs, last, err = domain.ChangeLog(start, end, count, lastId)
	} else {
		chs, last, err = domain.ServiceChangeLog(id, start, end, count, lastId)
	}
	if err != nil {
		return nil, errors.InternalServerError("com.HailoOSS.service.config.changelog", fmt.Sprintf("%v", err))
	}

	return &changelog.Response{
		Changes: changesToFullProto(chs),
		Last:    proto.String(last),
	}, nil
}
