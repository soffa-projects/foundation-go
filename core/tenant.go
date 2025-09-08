package f

import (
	"fmt"
	"io/fs"
	"net/http"
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
	err := detectTenant(c)
	if err != nil {
		return c.Error(err.Error(), ResponseOpt{Code: http.StatusBadRequest})
	}
	return nil
}

func TenantOptionalMiddleware(c Context) error {
	_ = detectTenant(c)
	return nil
}

func detectTenant(c Context) error {
	env := c.Env()
	tenantId := c.TenantId()
	if tenantId == "" {
		tenantId = c.Param("tenant")
	}
	if tenantId == "" {
		tenantId = c.QueryParam("tid")
	}
	if tenantId == "" {
		tenantId = c.Header("X-TenantId")
	}
	if tenantId == "" {
		tenantId = c.Host()
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
			return fmt.Errorf("INVALID_TENANT_001")
		}
		c.SetTenant(exists.ID)
	} else {
		return fmt.Errorf("INVALID_TENANT_002")
	}
	return nil
}
