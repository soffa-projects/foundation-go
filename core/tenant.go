package f

import (
	"io/fs"

	"github.com/soffa-projects/foundation-go/errors"
	"github.com/soffa-projects/foundation-go/h"
)

type TenantInput struct {
	Tenant string `param:"tenant" header:"X-TenantId" json:"-" validate:"required"`
}

type TenantProviderConfig struct {
	DefaultDatabaseURL string
	MigrationsFS       fs.FS
}

func TenantMiddleware(c Context) error {
	if c.TenantId() == "" {
		return errors.BadRequest("TENANT_REQUIRED_000")
	}
	return nil
}

func Authenticated(c Context) error {
	if c.Auth() == nil {
		return errors.Unauthorized("UNAUTHORIZED_000")
	}
	return nil
}

func PermissionMiddleware(permissions ...string) Middleware {
	return func(c Context) error {
		auth := c.Auth()
		if auth == nil {
			return errors.Unauthorized("UNAUTHORIZED_000")
		}
		if !h.ContainsAnyString(permissions, auth.Permissions) {
			return errors.Forbidden("FORBIDDEN_PERMISSION")
		}
		return nil
	}
}
