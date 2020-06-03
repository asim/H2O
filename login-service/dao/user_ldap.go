package dao

import (
	"regexp"

	"github.com/hailo-platform/H2O/login-service/domain"
	"github.com/hailo-platform/H2O/platform/multiclient"
	"github.com/hailo-platform/H2O/platform/server"
	"github.com/hailo-platform/H2O/service/config"

	readusersproto "github.com/hailo-platform/H2O/ldap-service/proto/readusers"
)

var (
	emailRegex = regexp.MustCompile(`^([a-zA-Z0-9._%+-]+)@([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})$`)
)

// isLDAPUser checks if the user ID is an LDAP user, a user is an LDAP user if:
//  - The user is an ADMIN user
//  - The uid is an email address
func IsLDAPUser(app domain.Application, uid string) (username, domain string, ok bool) {
	// Check application type
	if app != "ADMIN" {
		return "", "", false
	}

	// Check uid format
	matches := emailRegex.FindStringSubmatch(uid)
	if len(matches) != 3 {
		return "", "", false
	}

	whitelisted := config.AtPath("hailo", "ldap", "domains", matches[2]).AsBool()
	if !whitelisted {
		return matches[1], matches[2], false
	}

	return matches[1], matches[2], true
}

func readLDAPUsers(ids []string) ([]*domain.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	cl := multiclient.New().DefaultScopeFrom(server.Scoper())

	rsp := &readusersproto.Response{}
	cl.AddScopedReq(&multiclient.ScopedReq{
		Uid:      "ldap_login",
		Service:  "com.hailocab.service.ldap",
		Endpoint: "readusers",
		Req: &readusersproto.Request{
			Ids: ids,
		},
		Rsp: rsp,
	})

	if cl.Execute().AnyErrors() {
		return nil, cl.PlatformError("ldap_readusers")
	}

	users := make([]*domain.User, len(rsp.Users))
	for i, user := range rsp.Users {
		users[i] = ConvertLDAPUser(user)
	}

	return users, nil
}
