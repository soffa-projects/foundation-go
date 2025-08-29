package f

import (
	"context"
	"fmt"
	"net/http"

	"github.com/a-h/templ"
)

type Methods struct {
	GET    HandlerInit
	POST   HandlerInit
	DELETE HandlerInit
	PUT    HandlerInit
	PATCH  HandlerInit
}

type Router interface {
	Handler() http.Handler
	Listen(port int)
	Shutdown(ctx context.Context) error
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
	Env() Env
	Auth() *Authentication
	RealIP() string
	UserAgent() string
	TenantId() string
	SetTenant(tenantId string)
	//WithConnection(conn Connection) Context

	Bind(input any)
	ShouldBind(input any) error
	Param(value string) string
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
}

type Middleware = func(Context) error

type Route struct {
	Transactional bool
	Roles         []string
	Pre           []Middleware
	Handle        func(Context) any
	Authenticated bool
}
type HandlerInit = func(Env) Route

type Authentication struct {
	UserId     string
	Audience   []string
	Role       string
	Permission string
	Email      string
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
