package handler

import (
	"github.com/HailoOSS/login-service/domain"
	"github.com/HailoOSS/platform/errors"
	"github.com/HailoOSS/platform/server"
)

// authoriseAdmin checks we have ADMIN role
func authoriseAdmin(req *server.Request) errors.Error {
	if req.Auth().HasAccess("ADMIN.EXPERIMENT") {
		return nil
	}

	return errors.Forbidden(server.Name+".authorisation", "Permission denied (unauthorised role)")
}

// authoriseCreate tests if we can create this user
func authoriseCreate(req *server.Request, user *domain.User) errors.Error {
	if user.AnyAdminRoles() {
		// MUST be a real person with ADMIN
		if !req.Auth().IsAuth() || !req.Auth().AuthUser().HasRole("ADMIN") {
			return errors.Forbidden(server.Name+".authorisation", "Permission denied (unauthorised user)")
		}
		return nil
	}

	// not admin -- we'll allow service-to-service auth as well thus check via HasAccess
	if !req.Auth().HasAccess("ADMIN.CREATEUSER") {
		return errors.Forbidden(server.Name+".authorisation", "Permission denied (unauthorised role)")
	}

	return nil
}
