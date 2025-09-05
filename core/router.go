package f

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"time"

	"github.com/a-h/templ"
)

type RouterConfig struct {
	AllowOrigins  []string
	AssetsFS      fs.FS
	FaviconFS     fs.FS
	SessionSecret string
	SentryDSN     string
	Env           string
}

type Methods struct {
	GET    HandlerInit
	POST   HandlerInit
	DELETE HandlerInit
	PUT    HandlerInit
	PATCH  HandlerInit
}

type RouterGroup interface {
	GET(path string, handler HandlerInit)
	POST(path string, handler HandlerInit)
	DELETE(path string, handler HandlerInit)
	PUT(path string, handler HandlerInit)
	PATCH(path string, handler HandlerInit)
}

type Router interface {
	Init(env ApplicationEnv)
	Handler() http.Handler
	Listen(port int)
	Shutdown(ctx context.Context) error
	Group(path string, middleware ...Middleware) RouterGroup
	GET(path string, handler HandlerInit)
	POST(path string, handler HandlerInit)
	DELETE(path string, handler HandlerInit)
	PUT(path string, handler HandlerInit)
	PATCH(path string, handler HandlerInit)
	Use(middleware Middleware)
}

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
	Unauthorized(message string) HttpResponse
	Forbidden(message string) HttpResponse
	BadRequest(message string) HttpResponse
	NotFound(message string) HttpResponse
	Conflict(message string) HttpResponse
	Redirect(code int, url string) RedirectResponse

	File(data []byte, contentType string, filename string) HttpResponse
	Created(output any) HttpResponse
	NoContent() HttpResponse

	NewCsrfToken(duration time.Duration) (string, error)
	ValidateCsrfToken(token string) error
}

type Middleware = func(Context) error

type Route struct {
	Transactional bool
	Permissions   []string
	Pre           []Middleware
	Handle        func(Context) any
	Authenticated bool
}
type HandlerInit = func(ApplicationEnv) Route

type Authentication struct {
	UserId      string
	Audience    []string
	Permissions []string
	Email       string
	TenantId    string
}

type RedirectResponse struct {
	Code int
	Url  string
}

type HttpResponse struct {
	//error
	Code        int
	File        bool
	Template    templ.Component
	Data        any
	ContentType string
	Filename    string
}

func (e RedirectResponse) Error() string {
	return fmt.Sprintf("%d - %s", e.Code, e.Url)
}

func (e HttpResponse) Error() string {
	return fmt.Sprintf("%d - %s", e.Code, e.Data)
}
