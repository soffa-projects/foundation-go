package f

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/soffa-projects/foundation-go/log"
)

type TenantInput struct {
	Tenant string `param:"tenant" header:"X-TenantId" json:"-" validate:"required"`
}

type TenantProviderConfig struct {
	DefaultDatabaseURL string
	MigrationsFS       fs.FS
}

func TenantMiddleware(c Context) error {
	env := c.Env()
	tenantId := c.TenantId()
	if tenantId == "" {
		tenantId = c.Param("tenant")
	}
	if tenantId == "" {
		tenantId = c.Header("X-TenantId")
	}
	if tenantId != "" {
		if env.TenantProvider == nil {
			return fmt.Errorf("TENANT_PROVIDER_NOT_SET")
		}
		tenantId = strings.ToLower(tenantId)
		exists, err := env.TenantProvider.GetTenant(c, tenantId)
		if err != nil {
			return err
		}
		if exists == nil {
			log.Info("invalid tenant received: %s", tenantId)
			return c.BadRequest("INVALID_TENANT")
		}
		c.SetTenant(tenantId)
	} else {
		return c.BadRequest("TENANT_REQUIRED")
	}
	return nil
}
