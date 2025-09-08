package adapters

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/ztrue/tracerr"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	prettylogger "github.com/rdbell/echo-pretty-logger"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

const _authKey = "auth"
const _authTokenKey = "authToken"
const _tenantIdKey = "tenantId"
const _envKey = "env"

//const _connectionKey = "connection"

func NewEchoRouter(cfg *f.RouterConfig) f.Router {
	e := echo.New()
	e.Use(prettylogger.Logger)
	if cfg.Debug {
		e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
			LogLevel: 2,
			LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
				tracerr.PrintSourceColor(err, 2)
				return err
			},
		}))
	} else {
		e.Use(middleware.Recover())
	}
	e.Use(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestID())

	if cfg.AssetsFS != nil {
		e.StaticFS("/assets", cfg.AssetsFS)
	}
	if cfg.FaviconFS != nil {
		e.FileFS("/favicon.ico", "favicon.ico", cfg.FaviconFS)
	}
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	if cfg.SessionSecret != "" {
		e.Logger.Info("session secret found, enabling session middleware")
		e.Use(session.Middleware(sessions.NewCookieStore([]byte(cfg.SessionSecret))))
	}

	if cfg != nil && cfg.AllowOrigins != nil {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: cfg.AllowOrigins,
			AllowHeaders: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		}))
	}
	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:         cfg.SentryDSN,
			Environment: cfg.Env,
		}); err != nil {
			log.Fatal("Sentry initialization failed: %v\n", err)
		}

		e.Use(sentryecho.New(sentryecho.Options{}))
		log.Info("[echo] sentry middle initialized successfully")
	}

	return &routerImpl{
		internal: e,
	}
}

func (r *routerImpl) Init(env f.ApplicationEnv) {
	r.env = env

	r.internal.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(_tenantIdKey, "")
			c.Set(_authKey, (*f.Authentication)(nil))
			c.Set(_envKey, env)
			authToken := ""
			authz := c.Request().Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				authToken = authz[len("bearer "):]
			}
			if authToken != "" {
				if env.TokenProvider != nil {
					token, err := env.TokenProvider.Verify(authToken)
					if err == nil {
						sub, _ := token.Subject()
						aud, _ := token.Audience()
						var email string
						var tenantId string

						permissions := h.GetClaimValues(token, "permissions", "permission", "grant", "grants", "roles", "role")
						_ = token.Get("email", &email)
						_ = token.Get("tenantId", &tenantId)
						//c.Set("authToken", authToken)
						auth := &f.Authentication{
							UserId:      sub,
							Audience:    aud,
							Permissions: permissions,
							Email:       email,
							TenantId:    tenantId,
						}
						c.Set(_authKey, auth)
						if tenantId != "" {
							c.Set(_tenantIdKey, tenantId)
						}
					}
				}
				c.Set(_authTokenKey, authToken)
			}
			return next(c)
		}
	})
}

func (r *routerImpl) Handler() http.Handler {
	return r.internal
}

func (r *routerImpl) Listen(port int) {
	if port == 0 {
		port = 8080
	}
	err := r.internal.Start(fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal("failed to start server: %v", err)
	}
}

func (r *routerImpl) Shutdown(ctx context.Context) error {
	return r.internal.Shutdown(ctx)
}

/*
type Ctx struct {
	context.Context
	//echo.Context
	internal echo.Context
	//Context  context.Context
	//UserID    string
	Auth      *Authentication
	RealIP    string
	UserAgent string
	Env       *Env
	Tx        DB
	AuthToken string
	TenantId  string
	Tenant    any
}
*/

/*
func (c *ctxImpl) Unwrap() context.Context {
	return c.internal.Request().Context()
}

func (c *ctxImpl) Get(key string) any {
	return c.internal.Get(key)
}



func (c *ctxImpl) Request() *http.Request {
	return c.internal.Request()
}

func (c *ctxImpl) Response() http.ResponseWriter {
	return c.internal.Response()
}

func (c *ctxImpl) SetCookie(name string, value string, duration time.Duration) {
	cookie := new(http.Cookie)
	cookie.Name = name
	cookie.Value = value
	cookie.Expires = time.Now().Add(duration)
	c.internal.SetCookie(cookie)
}

func (c *ctxImpl) GetCookie(name string) string {
	cookie, err := c.internal.Request().Cookie(name)
	if err != nil {
		log.Error("failed to get cookie: %v", err)
		return ""
	}
	return cookie.Value
}


*/

type routerImpl struct {
	f.Router
	internal *echo.Echo
	env      f.ApplicationEnv
}

type operationContextImpl struct {
	inputSchema any
	env         f.ApplicationEnv
	router      echo.Context
	tenantCnx   f.Connection
	defaultCnx  f.Connection
	context.Context
}

func (r *operationContextImpl) Env() f.ApplicationEnv {
	return r.env
}

func (r *operationContextImpl) Redirect(url string, status ...int) error {
	st := http.StatusFound
	if len(status) > 0 {
		st = status[0]
	}
	return r.router.Redirect(st, url)
}

func (r *operationContextImpl) Render(template templ.Component, status ...int) error {
	html, err := h.RenderTempl(r.Context, template)
	if err != nil {
		return err
	}
	st := http.StatusOK
	if len(status) > 0 {
		st = status[0]
	}
	return r.router.HTML(st, html)
}

func (c *operationContextImpl) Auth() *f.Authentication {
	value := c.router.Get(_authKey)
	if value == nil {
		return nil
	}
	return value.(*f.Authentication)
}

func (c *operationContextImpl) AuthToken() string {
	value := c.router.Get(_authTokenKey)
	if value == nil {
		return ""
	}
	return value.(string)
}

func (c *operationContextImpl) RealIP() string {
	return c.router.RealIP()
}

func (c *operationContextImpl) UserAgent() string {
	return c.router.Request().UserAgent()
}

func (r *operationContextImpl) Error(error string, opt ...f.ResponseOpt) error {
	st := http.StatusInternalServerError
	for _, o := range opt {
		if o.Code != 0 {
			st = o.Code
		}
	}
	response := r.router.Response()
	requestt := r.router.Request()

	return r.router.JSON(st, map[string]any{
		"requestId": response.Header().Get(echo.HeaderXRequestID),
		"timestamp": time.Now().Format(time.RFC3339),
		"uri":       requestt.URL.Path,
		"error":     error,
		"success":   false,
	})
}

func (r *routerImpl) AddOperation(operation f.Operation) {
	if operation.Transport.Http.Path == "" {
		return
	}
	httpTransport := operation.Transport.Http
	methods := []string{http.MethodGet}
	if httpTransport.Methods != nil {
		methods = httpTransport.Methods
	} else if httpTransport.Method != "" {
		methods = []string{httpTransport.Method}
	}
	path := httpTransport.Path
	env := r.env

	handler := func(c echo.Context) error {

		ctx := &operationContextImpl{
			router:      c,
			inputSchema: operation.Schemas.Input,
			env:         r.env,
			Context:     c.Request().Context(),
		}

		auth := ctx.Auth()

		for _, middleware := range operation.Middlewares {
			if err := middleware(ctx); err != nil {
				return formatError(ctx.router, err)
			}
		}

		tenantId := ctx.TenantId()

		if operation.Authenticated && auth == nil {
			return ctx.Error("unauthorized_no_auth_01", f.ResponseOpt{Code: http.StatusUnauthorized})
		}

		if operation.Permissions != nil && auth == nil {
			return ctx.Error("unauthorized_no_auth_02", f.ResponseOpt{Code: http.StatusUnauthorized})
		}

		if operation.Permissions != nil && auth != nil {
			if !h.ContainsAnyString(operation.Permissions, auth.Permissions) {
				return ctx.Error("forbidden_grants", f.ResponseOpt{Code: http.StatusForbidden})
			}
		}

		if env.DS != nil {
			ctx.defaultCnx = env.DS.DefaultConnection()
			if ctx.defaultCnx != nil {
				tx, err := ctx.defaultCnx.Tx(ctx)
				if err != nil {
					return err
				}
				ctx.defaultCnx = tx
				ctx.Context = context.WithValue(ctx.Context, f.DefaultCnxKey{}, ctx.defaultCnx)
			}
			if tenantId != "" {
				ctx.tenantCnx = env.DS.Connection(tenantId)
				if ctx.tenantCnx != nil {
					tx, err := ctx.tenantCnx.Tx(ctx)
					if err != nil {
						return err
					}
					ctx.tenantCnx = tx
					ctx.Context = context.WithValue(ctx.Context, f.TenantCnxKey{}, ctx.tenantCnx)
				}
			}
		}

		ctx.Context = context.WithValue(ctx.Context, f.TenantKey{}, tenantId)
		ctx.Context = context.WithValue(ctx.Context, f.AuthenticationKey{}, auth)

		response, err := operation.Handle(ctx)
		if err != nil {
			return formatError(ctx.router, err)
		}

		return formatResponse(ctx.router, response)
	}

	for _, method := range methods {
		switch method {
		case http.MethodGet:
			r.internal.GET(path, handler)
		case http.MethodPost:
			r.internal.POST(path, handler)
		case http.MethodDelete:
			r.internal.DELETE(path, handler)
		case http.MethodPut:
			r.internal.PUT(path, handler)
		case http.MethodPatch:
			r.internal.PATCH(path, handler)
		default:
			log.Fatal("invalid http method: %s", method)
		}
	}
}

func formatError(ctx echo.Context, err error) error {
	code := http.StatusInternalServerError
	errorMessage := err.Error()
	if errors.Is(err, f.OperationError{}) {
		code = http.StatusBadRequest
		er := err.(f.OperationError)
		if er.Code != 0 {
			code = er.Code
		}
	}
	return ctx.JSON(code, map[string]any{
		"requestId": ctx.Response().Header().Get(echo.HeaderXRequestID),
		"timestamp": time.Now().Format(time.RFC3339),
		"uri":       ctx.Request().URL.Path,
		"error":     errorMessage,
		"success":   false,
	})
	// return tracerr.Wrap(err)
}

func formatResponse(ctx echo.Context, res any) error {

	var wrapped f.Response

	if _, ok := res.(f.Response); ok {
		wrapped = res.(f.Response)
	} else {
		wrapped = f.Response{
			Data: res,
		}
	}

	st := http.StatusOK
	if wrapped.Data == nil {
		st = http.StatusNoContent
	}
	response := ctx.Response()
	contentType := "application/json"
	if wrapped.Code != 0 {
		st = wrapped.Code
	}
	if wrapped.ContentType != "" {
		contentType = wrapped.ContentType
	}
	if contentType == "application/json" {
		return ctx.JSON(st, wrapped.Data)
	}
	response.Header().Set("Content-Type", contentType)
	response.WriteHeader(st)
	response.Write([]byte(wrapped.Data.(string)))
	return nil
}

func (r *routerImpl) MCP(path string, handler http.Handler) {
	wrapped := echo.WrapHandler(handler)
	r.internal.POST(path, wrapped)
	r.internal.GET(path, wrapped)
	r.internal.Any(path+"/*", wrapped)
}

func (c *operationContextImpl) Param(value string) string {
	return c.router.Param(value)
}

func (c *operationContextImpl) QueryParam(value string) string {
	return c.router.QueryParam(value)
}
func (c *operationContextImpl) FormFile(field string) (io.ReadCloser, error) {
	file, err := c.router.FormFile(field)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, errors.New("err_file_required")
	}
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	return src, nil
}

func (c *operationContextImpl) Header(value string) string {
	return c.router.Request().Header.Get(value)
}

func (c *operationContextImpl) Set(key string, value any) {
	c.router.Set(key, value)
}

func (c *operationContextImpl) Bind(input any) error {
	err := c.ShouldBind(input)
	return err
}

func (c *operationContextImpl) Host() string {
	return strings.ToLower(c.router.Request().Host)
}

func (c *operationContextImpl) SetTenant(tenantId string) {
	c.Set(_tenantIdKey, tenantId)
	c.Context = context.WithValue(c.Context, f.TenantKey{}, tenantId)
}

/*
	func (c *ctxImpl) WithValue(key, value any) f.Context {
		return &ctxImpl{
			Context:  context.WithValue(c.Context, key, value),
			internal: c.internal,
			env:      c.env,
		}
	}
*/
func (c *operationContextImpl) TenantId() string {
	value := c.router.Get(_tenantIdKey)
	if value == nil {
		return ""
	}
	return value.(string)
}

func (c *operationContextImpl) ShouldBind(input any) error {
	binder := &echo.DefaultBinder{}
	if err := binder.BindHeaders(c.router, input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := binder.BindQueryParams(c.router, input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := binder.BindPathParams(c.router, input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := binder.BindBody(c.router, input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

/*
func (c *ctxImpl) SetSession(key string, value string, maxAge int) error {
	sess, err := session.Get("session", c.internal)
	if err != nil {
		return err
	}
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
	}
	sess.Values[key] = value
	return sess.Save(c.internal.Request(), c.internal.Response())
}

func (c *ctxImpl) File(data []byte, contentType string, filename string) f.HttpResponse {
	// Return the PDF data
	return f.HttpResponse{
		Code:        http.StatusOK,
		File:        true,
		Data:        data,
		ContentType: contentType,
		Filename:    filename,
	}
}

*/
