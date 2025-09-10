package f

import (
	"context"
	"io/fs"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

type App interface {
	Start(port int)
	Shutdown(ctx context.Context)
	Router() Router
}

type AppInfo struct {
	Name      string
	Version   string
	PublicURL string
}

type AppConfig = any

type HttpRouter interface {
	GET(path string, handler func(c HttpContext) error, middlewares ...Middleware)
	POST(path string, handler func(c HttpContext) error, middlewares ...Middleware)
	DELETE(path string, handler func(c HttpContext) error, middlewares ...Middleware)
	PUT(path string, handler func(c HttpContext) error, middlewares ...Middleware)
	PATCH(path string, handler func(c HttpContext) error, middlewares ...Middleware)
}

type McpRouter interface {
	Add(operation MCP)
	IsEmpty() bool
}

type MCP struct {
	Name         string
	Desc         string
	InputSchema  any
	OutputSchema any
	Handle       func(c McpContext) (any, error)
}

type Feature struct {
	Name      string
	FS        fs.FS
	DependsOn []Feature
	Init      func(c InitContext)
}

type InitContext struct {
	Config AppConfig
	Router HttpRouter
	MCP    McpRouter
}

/*
const (
	StatusOK             = http.StatusOK
	StatusCreated        = http.StatusCreated
	StatusAccepted       = http.StatusAccepted
	StatusNoContent      = http.StatusNoContent
	StatusBadRequest     = http.StatusBadRequest
	StatusUnauthorized   = http.StatusUnauthorized
	StatusForbidden      = http.StatusForbidden
	StatusNotFound       = http.StatusNotFound
	StatusTechnicalError = http.StatusInternalServerError
	StatusConflict       = http.StatusConflict
)
*/

type Context interface {
	context.Context
	TenantId() string
	Auth() *Authentication
	AuthToken() string
	SetTenant(tenantId string)
	RemoteAddr() string
}

type HttpContext interface {
	Context

	UserAgent() string
	Param(value string) string
	QueryParam(value string) string
	Header(value string) string
	Host() string
	Bind(value any) error
	//
	JSON(status int, data any) error
	Redirect(status int, url string) error
	HTML(status int, content string) error
	NoContent() error
}

type McpContext interface {
	Context
	Structured(data any) any
}
