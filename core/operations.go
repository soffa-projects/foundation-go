package f

import (
	"context"

	"github.com/a-h/templ"
)

type Resource struct {
	MimeType string
	Content  any
}

type ResourceList struct {
	Items []Resource
}

type Result struct {
	Success     bool
	Code        int
	Type        string
	Message     string
	Error       error
	Data        any
	ContentType string
}

type ResponseOpt struct {
	Code        int
	ContentType string
}

type Context interface {
	context.Context
	Env() ApplicationEnv
	RealIP() string
	UserAgent() string
	Send(value any, opt ...ResponseOpt) error
	Render(template templ.Component, status ...int) error
	Redirect(url string, status ...int) error
	Error(error string, opt ...ResponseOpt) error
	TenantId() string
	Param(value string) string
	QueryParam(value string) string
	Header(value string) string
	Host() string
	SetTenant(tenantId string)
	Auth() *Authentication
	AuthToken() string
	Bind(value any) error
}

type Operation struct {
	Context       Context
	Name          string
	Description   string
	InputSchema   any
	OutputSchema  any
	Handle        func(ctx Context) error
	Method        string
	Methods       []string
	Path          string
	Middlewares   []Middleware
	Authenticated bool
	Permissions   []string
}

type OperationError struct {
	Code int
	Err  error
}

func (e *OperationError) Error() string {
	return e.Err.Error()
}
