package adapters

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/thoas/go-funk"
	"github.com/ztrue/tracerr"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	prettylogger "github.com/rdbell/echo-pretty-logger"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/errors"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

const _authKey = "auth"
const _authTokenKey = "authToken"
const _tenantIdKey = "tenantId"

type EchoRouterConfig struct {
	Debug         bool
	PublicFS      fs.FS
	SessionSecret string
	AllowOrigins  []string
	SentryDSN     string
	Env           string
	TokenProvider f.TokenProvider
}

func NewEchoRouter(cfg EchoRouterConfig) f.Router {
	e := echo.New()
	e.Use(prettylogger.Logger)
	/*
		if cfg.Debug {
			e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
				LogLevel: 2,
				LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
					tracerr.PrintSourceColor(tracerr.Wrap(err), 1)
					return err
				},
			}))
		} else {

		}*/
	e.Use(middleware.Recover())
	e.Use(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestID())

	if cfg.PublicFS != nil {
		e.FileFS("/favicon.ico", "favicon.ico", cfg.PublicFS)
		e.StaticFS("/assets", cfg.PublicFS)
	}
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	if cfg.SessionSecret != "" {
		e.Logger.Info("session secret found, enabling session middleware")
		e.Use(session.Middleware(sessions.NewCookieStore([]byte(cfg.SessionSecret))))
	}

	if !funk.IsEmpty(cfg.AllowOrigins) {
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

	// Tenant middleware

	return &routerImpl{
		internal:      e,
		tokenProvider: cfg.TokenProvider,
	}
}

// ------------------------------------------------------------------------------------------------------------------
// ECHO ROUTER IMPL
// ------------------------------------------------------------------------------------------------------------------

type routerImpl struct {
	f.Router
	internal      *echo.Echo
	tokenProvider f.TokenProvider
	dataSource    f.DataSource
}

func (r *routerImpl) Init() {

	ds := f.Lookup[f.DataSource]()
	if ds != nil {
		r.dataSource = *ds
	}

	r.internal.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(_tenantIdKey, "")
			c.Set(_authKey, (*f.Authentication)(nil))
			//c.Set(_envKey, env)
			authToken := ""
			authz := c.Request().Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				authToken = authz[len("bearer "):]
			}

			tenantId := ""

			if tenantId == "" {
				tenantId = c.Param("tenant")
			}
			if tenantId == "" {
				tenantId = c.QueryParam("tid")
			}
			if tenantId == "" {
				tenantId = c.Request().Header.Get("X-TenantId")
			}
			if tenantId == "" {
				value := c.Request().Host
				//TODO: check if it's a valid domain name
				if h.IsDomainName(value) {
					tenantId = value
				}
			}

			if authToken != "" {
				if r.tokenProvider != nil {
					token, err := r.tokenProvider.Verify(authToken)
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
					}
				}
				c.Set(_authTokenKey, authToken)
			}

			if tenantId != "" {
				tenantId = strings.ToLower(tenantId)
				tenantProvider := f.Lookup[f.TenantProvider]()
				if tenantProvider != nil {
					exists, err := (*tenantProvider).GetTenant(c.Request().Context(), tenantId)
					if err != nil {
						return err
					}
					if exists == nil {
						log.Info("invalid tenant received: %s", tenantId)
					} else {
						c.Set(_tenantIdKey, tenantId)
					}
				} else {
					c.Set(_tenantIdKey, tenantId)
				}
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

func (r *routerImpl) GET(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.GET(path, r.wrapHandler(handler, middlewares...))
}

func (r *routerImpl) POST(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.POST(path, r.wrapHandler(handler, middlewares...))
}

func (r *routerImpl) DELETE(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.DELETE(path, r.wrapHandler(handler, middlewares...))
}

func (r *routerImpl) PUT(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.PUT(path, r.wrapHandler(handler, middlewares...))
}

func (r *routerImpl) PATCH(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.PATCH(path, r.wrapHandler(handler, middlewares...))
}

func (r *routerImpl) wrapHandler(handler func(c f.HttpContext) error, middlewares ...f.Middleware) echo.HandlerFunc {
	return func(c echo.Context) error {

		ctx := &httpContextImpl{
			internal: c,
			Context:  c.Request().Context(),
		}

		defer func() {

			defaultCnx := ctx.Value(f.DefaultCnxKey{})
			tenantCnx := ctx.Value(f.TenantCnxKey{})

			if err := recover(); err != nil {
				tracerr.PrintSourceColor(tracerr.Wrap(err.(error)), 1)
				if defaultCnx != nil {
					defaultCnx.(f.Connection).Rollback()
				}
				if tenantCnx != nil {
					tenantCnx.(f.Connection).Rollback()
				}
				formatError(ctx.internal, err.(error), http.StatusInternalServerError)
			} else {
				if defaultCnx != nil {
					defaultCnx.(f.Connection).Commit()
				}
				if tenantCnx != nil {
					tenantCnx.(f.Connection).Commit()
				}
			}
		}()

		auth := ctx.Auth()

		for _, middleware := range middlewares {
			if err := middleware(ctx); err != nil {
				return formatError(ctx.internal, err, http.StatusBadRequest)
			}
		}

		tenantId := ctx.TenantId()

		if r.dataSource != nil {
			defaultCnx := r.dataSource.DefaultConnection()
			if defaultCnx != nil {
				tx, err := defaultCnx.Tx(ctx)
				if err != nil {
					return err
				}
				defaultCnx = tx
				ctx.Context = context.WithValue(ctx.Context, f.DefaultCnxKey{}, defaultCnx)
			}
			if ctx.TenantId() != "" {
				tenantCnx := r.dataSource.Connection(ctx.TenantId())
				if tenantCnx != nil {
					tx, err := tenantCnx.Tx(ctx)
					if err != nil {
						return err
					}
					tenantCnx = tx
					ctx.Context = context.WithValue(ctx.Context, f.TenantCnxKey{}, tenantCnx)
				}
			}
		}

		ctx.Context = context.WithValue(ctx.Context, f.TenantKey{}, tenantId)
		ctx.Context = context.WithValue(ctx.Context, f.AuthenticationKey{}, auth)

		err := handler(ctx)

		if err != nil {
			return formatError(ctx.internal, err, 0)
		}

		return nil
	}
}

// ------------------------------------------------------------------------------------------------------------------
// HTTP CONTEXT IMPL
// ------------------------------------------------------------------------------------------------------------------

type httpContextImpl struct {
	context.Context
	internal echo.Context
}

func (c *httpContextImpl) Auth() *f.Authentication {
	value := c.internal.Get(_authKey)
	if value == nil {
		return nil
	}
	return value.(*f.Authentication)
}

func (h *httpContextImpl) Value(key any) any {
	return h.Context.Value(key)
}

func (c *httpContextImpl) AuthToken() string {
	value := c.internal.Get(_authTokenKey)
	if value == nil {
		return ""
	}
	return value.(string)
}

func (c *httpContextImpl) TenantId() string {
	return c.internal.Get(_tenantIdKey).(string)
}

func (c *httpContextImpl) Param(value string) string {
	return c.internal.Param(value)
}

func (c *httpContextImpl) QueryParam(value string) string {
	return c.internal.QueryParam(value)
}

func (c *httpContextImpl) Header(value string) string {
	return c.internal.Request().Header.Get(value)
}

func (c *httpContextImpl) Host() string {
	return c.internal.Request().Host
}

func (c *httpContextImpl) Bind(value any) error {
	err := c.ShouldBind(value)
	return err
}

func (c *httpContextImpl) ShouldBind(input any) error {
	binder := &echo.DefaultBinder{}
	if err := binder.BindHeaders(c.internal, input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := binder.BindQueryParams(c.internal, input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := binder.BindPathParams(c.internal, input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if err := binder.BindBody(c.internal, input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func (c *httpContextImpl) RemoteAddr() string {
	return c.internal.RealIP()
}

func (c *httpContextImpl) UserAgent() string {
	return c.internal.Request().UserAgent()
}

func (c *httpContextImpl) EM(tenantId ...string) f.Connection {
	defaultCnx := c.Value(f.DefaultCnxKey{})
	tenantCnx := c.Value(f.TenantCnxKey{})
	if tenantCnx != nil {
		if len(tenantId) > 0 && tenantId[0] == _defaultTenantId && defaultCnx != nil {
			return defaultCnx.(f.Connection)
		}
		return tenantCnx.(f.Connection)
	}
	if defaultCnx != nil {
		return defaultCnx.(f.Connection)
	}
	return nil
}

func (c *httpContextImpl) JSON(status int, data any) error {
	return c.internal.JSON(status, data)
}

func (c *httpContextImpl) HTML(status int, content string) error {
	return c.internal.HTML(http.StatusOK, content)
}

func (c *httpContextImpl) NoContent() error {
	return c.internal.NoContent(http.StatusNoContent)
}

func (c *httpContextImpl) Render(status int, template templ.Component) error {
	html, err := h.RenderTempl(c.internal.Request().Context(), template)
	if err != nil {
		return err
	}
	return c.internal.HTML(status, html)
}

func (c *httpContextImpl) Redirect(status int, url string) error {
	return c.internal.Redirect(status, url)
}

func (c *httpContextImpl) SetTenant(tenantId string) {
	c.internal.Set(_tenantIdKey, tenantId)
	c.Context = context.WithValue(c.Context, f.TenantKey{}, tenantId)
}

func formatError(ctx echo.Context, err error, code int) error {
	status := code
	if errors.Is(err, errors.CustomError{}) {
		status = err.(errors.CustomError).Code
	}
	errorMessage := err.Error()
	return ctx.JSON(status, map[string]any{
		"requestId": ctx.Response().Header().Get(echo.HeaderXRequestID),
		"timestamp": time.Now().Format(time.RFC3339),
		"uri":       ctx.Request().URL.Path,
		"error":     errorMessage,
		"success":   false,
	})
	// return tracerr.Wrap(err)
}

func (r *routerImpl) MCP(path string, handler http.Handler) {
	wrapped := echo.WrapHandler(handler)
	r.internal.POST(path, wrapped)
	r.internal.GET(path, wrapped)
	r.internal.Any(path+"/*", wrapped)
}

/*


func (r *routerImpl) AddOperation(operation f.Operation) {
	if operation.Http.Path == "" {
		return
	}
	httpTransport := operation.Http
	methods := []string{http.MethodGet}
	if httpTransport.Methods != nil {
		methods = httpTransport.Methods
	} else if httpTransport.Method != "" {
		methods = []string{httpTransport.Method}
	}
	path := httpTransport.Path
	//datasource := f.Resolve[f.DataSource]()

	handler := func(c echo.Context) error {


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


func formatResponse(ctx echo.Context, res f.Response) error {

	if res.Err != nil {
		log.Error("http-error: %v", res.Err)
		return ctx.JSON(http.StatusInternalServerError, map[string]any{
			"requestId": ctx.Response().Header().Get(echo.HeaderXRequestID),
			"timestamp": time.Now().Format(time.RFC3339),
			"uri":       ctx.Request().URL.Path,
			"error":     res.Err.Error(),
			"success":   false,
		})
	}
	data := res.Data
	status := res.Code
	if status == 0 {
		status = http.StatusOK
		if data == nil {
			status = http.StatusNoContent
		}
	}
	contentType := "application/json"
	if res.Opts != nil {
		for _, opt := range res.Opts {
			if value, ok := opt.(f.HttpOpt); ok {
				if value.ContentType != "" {
					contentType = value.ContentType
				}
				if value.Redirect {
					return ctx.Redirect(value.Code, data.(string))
				}
			}
		}
	}

	if contentType == "application/json" {
		return ctx.JSON(status, data)
	}

	if contentType == "text/html" {
		if tpl, ok := data.(templ.Component); ok {
			html, err := h.RenderTempl(ctx.Request().Context(), tpl)
			if err != nil {
				return err
			}
			return ctx.HTML(status, html)
		}

		if str, ok := data.(string); ok {
			return ctx.HTML(status, str)
		}

	}

	log.Warn("unsupported content type: %s", contentType)

	return ctx.JSON(http.StatusInternalServerError, map[string]any{
		"requestId": ctx.Response().Header().Get(echo.HeaderXRequestID),
		"timestamp": time.Now().Format(time.RFC3339),
		"uri":       ctx.Request().URL.Path,
		"error":     fmt.Sprintf("UNSUPPORTED_CONTENT_TYPE: %s", contentType),
		"success":   false,
	})
}

func (c *operationContextImpl) Set(key any, value any) {
	c.Context = context.WithValue(c.Context, key, value)
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

func (c *operationContextImpl) Bind(input any) error {
	err := c.ShouldBind(input)
	return err
}

func (c *operationContextImpl) Host() string {
	return strings.ToLower(c.router.Request().Host)
}


*/

/*
	func (c *ctxImpl) WithValue(key, value any) f.Context {
		return &ctxImpl{
			Context:  context.WithValue(c.Context, key, value),
			internal: c.internal,
			env:      c.env,
		}
	}

func (c *operationContextImpl) TenantId() string {
	value := c.router.Get(_tenantIdKey)
	if value == nil {
		return ""
	}
	return value.(string)
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
