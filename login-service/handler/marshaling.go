package handler

import (
	"time"

	"github.com/HailoOSS/protobuf/proto"

	"github.com/HailoOSS/login-service/domain"
	protoep "github.com/HailoOSS/login-service/proto"
	protosession "github.com/HailoOSS/login-service/proto/listsessions"
	protologin "github.com/HailoOSS/login-service/proto/readlogin"
	protouser "github.com/HailoOSS/login-service/proto/readuser"
)

func protoToTime(t *int64, def time.Time) time.Time {
	if t == nil {
		return def
	}
	return time.Unix(*t, 0)
}

func stringsToIds(a []string) []domain.Id {
	ret := make([]domain.Id, 0)
	for _, v := range a {
		if len(v) > 0 {
			ret = append(ret, domain.Id(v))
		}
	}
	return ret
}

// protoToEndpointAuth marshals proto -> domain for "endpoint auth"
func protoToEndpointAuth(prep *protoep.Endpoint) []*domain.EndpointAuth {
	ret := make([]*domain.EndpointAuth, 0)

	for _, grant := range prep.GetGranted() {
		ret = append(ret, &domain.EndpointAuth{
			ServiceName:    prep.GetService(),
			EndpointName:   prep.GetEndpoint(),
			AllowedService: grant.GetName(),
			Role:           grant.GetRole(),
		})
	}

	return ret
}

func protoToMap(kvs []*protoep.KeyValue) map[string]string {
	ret := make(map[string]string)
	for _, kv := range kvs {
		ret[kv.GetKey()] = kv.GetValue()
	}
	return ret
}

// userToProto marshals domain users -> proto
func userToProto(u *domain.User) *protouser.Response {
	return &protouser.Response{
		Application:      proto.String(string(u.App)),
		Uid:              proto.String(u.Uid),
		Ids:              idsToStrings(u.Ids),
		CreatedTimestamp: timeToProto(u.Created),
		Roles:            u.Roles,
		PasswordChangeTimestamp: timeToProto(u.PasswordChange),
		AccountExpirationDate:   proto.String(string(u.AccountExpirationDate)),
	}
}

func timeToProto(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}
	u := t.Unix()
	return &u
}

// loginsToProto marshals login domain -> proto
func loginsToProto(logins []*domain.Login) []*protologin.Login {
	rsp := make([]*protologin.Login, len(logins))
	for i, login := range logins {
		rsp[i] = &protologin.Login{
			Application:       proto.String(string(login.App)),
			Uid:               proto.String(login.Uid),
			LoggedInTimestamp: timeToProto(login.LoggedIn),
			Mech:              proto.String(login.AuthMechanism),
			DeviceType:        proto.String(login.DeviceType),
			Meta:              mapToProto(login.Meta),
		}
	}

	return rsp
}

// sessionToProto marshals a session -> proto
func sessionToProto(session *domain.Session) *protosession.Session {
	return &protosession.Session{
		CreatedTimestamp: timeToProto(session.Token.Created),
		Mech:             proto.String(session.Token.AuthMechanism),
		DeviceType:       proto.String(session.Token.DeviceType),
	}
}

func mapToProto(m map[string]string) []*protoep.KeyValue {
	kvs := make([]*protoep.KeyValue, 0)
	for k, v := range m {
		kvs = append(kvs, &protoep.KeyValue{
			Key:   proto.String(k),
			Value: proto.String(v),
		})
	}
	return kvs
}

func idsToStrings(a []domain.Id) []string {
	ret := make([]string, len(a))
	for i, v := range a {
		ret[i] = string(v)
	}
	return ret
}
