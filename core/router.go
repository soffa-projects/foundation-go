package f

import (
	"context"
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
}

type Context interface {
	context.Context
	AuthToken() string
	Env() Env
	Auth() *Authentication
	RealIP() string
	UserAgent() string
	TenantId() string
	// HttpResponse
	Unauthorized(message string) HttpResponse
	BadRequest(message string) HttpResponse
	NotFound(message string) HttpResponse
	Conflict(message string) HttpResponse
	Redirect(code int, url string) error
	Bind(input any) error
	Param(value string) string
	Set(key string, value any)
	File(data []byte, contentType string, filename string) HttpResponse
	Created(output any) HttpResponse
}

type Middleware = func(Context) error

type Route struct {
	Transactional bool
	PreAuthorize  func(Context) (bool, error)
	Handle        func(Context) any
}
type HandlerInit = func(Env) Route

type Authentication struct {
	UserId     string
	Audience   []string
	Permission string
	Email      string
}

type HttpResponse struct {
	error
	Code        int
	File        bool
	Template    templ.Component
	Data        any
	ContentType string
	Filename    string
}
