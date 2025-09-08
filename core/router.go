package f

import (
	"context"
	"io/fs"
	"net/http"
)

type RouterConfig struct {
	AllowOrigins  []string
	AssetsFS      fs.FS
	FaviconFS     fs.FS
	SessionSecret string
	SentryDSN     string
	Env           string
	Debug         bool
}

type Router interface {
	Init(env ApplicationEnv)
	Handler() http.Handler
	Listen(port int)
	Shutdown(ctx context.Context) error
	MCP(path string, handler http.Handler)
	Use(middleware Middleware)
	AddOperation(operation Operation)
}

/*
type Context interface {
	context.Context
	AuthToken() string
	Env() ApplicationEnv
	Auth() *Authentication
	RealIP() string
	UserAgent() string
	TenantId() string
	Host() string
	SetTenant(tenantId string)
	Request() *http.Request
	Response() http.ResponseWriter
	//WithConnection(conn Connection) Context

	SetCookie(name string, value string, duration time.Duration)
	GetCookie(name string) string

	Bind(input any)
	FormFile(key string) (io.ReadCloser, error)
	ShouldBind(input any) error
	Param(value string) string
	QueryParam(value string) string
	Header(value string) string
	Set(key string, value any)
	Get(key string) any
	WithValue(key, value any) Context
	// HttpResponse

	NewCsrfToken(duration time.Duration) (string, error)
	ValidateCsrfToken(token string) error
}*/

type Middleware = func(Context) error

type Authentication struct {
	UserId      string
	Audience    []string
	Permissions []string
	Email       string
	TenantId    string
}
