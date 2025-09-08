package adapters

import (
	"context"
)

type Resource struct {
	MimeType string
	Content  any
}

type ResourceList struct {
	Items []Resource
}

type Response struct {
	Code        int
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
	RealIP() string
	UserAgent() string
	//Send(value any, opt ...ResponseOpt) error
	//Render(template templ.Component, status ...int) error
	//Redirect(url string, status ...int) error
	//Error(error string, opt ...ResponseOpt) error
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

type OperationFn func() Operation

type HttpTransport struct {
	Method  string
	Methods []string
	Path    string
}

type Transport struct {
	Http HttpTransport
	Mcp  bool
}

type Schemas struct {
	Input  any
	Output any
}

type Operation struct {
	Context       Context
	Name          string
	Description   string
	Schemas       Schemas
	Handle        func(ctx Context) (any, error)
	Transport     Transport
	Middlewares   []Middleware
	Authenticated bool
	Permissions   []string
}

type OperationError struct {
	Code int
	Err  error
}

func (e OperationError) Error() string {
	return e.Err.Error()
}
