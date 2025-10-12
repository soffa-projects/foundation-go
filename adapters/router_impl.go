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
const _idemPotencyKey = "idempotencyKey"

type EchoRouterConfig struct {
	Debug          bool
	PublicFS       fs.FS
	SessionSecret  string
	AllowOrigins   []string
	SentryDSN      string
	Env            string
	TokenProvider  f.TokenProvider
	TenantProvider f.TenantProvider
	AuthProvider   f.AuthProvider
	DataSource     f.DataSource
}

func NewEchoRouter(cfg EchoRouterConfig) f.Router {
	e := echo.New()

	// Only use pretty logger in non-test environments for cleaner test output
	if cfg.Env != "test" {
		e.Use(prettylogger.Logger)
	}

	if cfg.Debug {
		e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
			LogLevel: 1,
			LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
				tracerr.PrintSourceColor(tracerr.Wrap(err), 3)
				return err
			},
		}))
	}

	e.Use(middleware.Recover())
	e.Use(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestID())

	if cfg.PublicFS != nil {

		e.FileFS("/favicon.ico", "favicon.ico", cfg.PublicFS)
		//e.StaticFS("/assets", cfg.PublicFS)

		assets := e.Group("/assets")
		assets.Use(middleware.Gzip())
		assets.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				return next(c)
			}
		})
		assets.StaticFS("/", cfg.PublicFS)

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
			log.Error("Sentry initialization failed: %v\n", err)
		} else {
			e.Use(sentryecho.New(sentryecho.Options{}))
			log.Info("[echo] sentry middle initialized successfully")
		}
	}

	// Tenant middleware

	return &routerImpl{
		internal:       e,
		tokenProvider:  cfg.TokenProvider,
		authProvider:   cfg.AuthProvider,
		tenantProvider: cfg.TenantProvider,
		ds:             cfg.DataSource,
	}
}

// ------------------------------------------------------------------------------------------------------------------
// ECHO ROUTER IMPL
// ------------------------------------------------------------------------------------------------------------------

type routerImpl struct {
	f.Router
	internal       *echo.Echo
	tokenProvider  f.TokenProvider
	authProvider   f.AuthProvider
	ds             f.DataSource
	tenantProvider f.TenantProvider
}

type groupRouterImpl struct {
	f.HttpRouter
	internal      *echo.Group
	tokenProvider f.TokenProvider
	ds            f.DataSource
	//tenantProvider f.TenantProvider
}

func (r *routerImpl) Init() {

	r.internal.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(_tenantIdKey, "")
			c.Set(_authKey, (*f.Authentication)(nil))
			//c.Set(_envKey, env)
			authToken := ""
			authz := c.Request().Header.Get("Authorization")
			idemPotencyKey := c.Request().Header.Get("Idempotency-Key")
			c.Set(_idemPotencyKey, idemPotencyKey)
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
				if h.IsDomainName(value) { //TODO: not always
					tenantId = value
				}
			}

			if authToken != "" {
				// ---
				authenticated := false
				if r.authProvider != nil {
					auth, err := r.authProvider.Authenticate(c.Request().Context(), authToken)
					if err == nil && auth != nil {
						c.Set(_authKey, auth)
						authenticated = true
						if auth.TenantId != "" {
							tenantId = auth.TenantId
						}
					}
					// ---
				}

				if !authenticated && r.tokenProvider != nil {
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
				if r.tenantProvider != nil {
					exists, err := (r.tenantProvider).GetTenant(c.Request().Context(), tenantId)
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

func (r *routerImpl) Listen(port int) error {
	if port == 0 {
		port = 8080
	}
	err := r.internal.Start(fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}
	return nil
}

func (r *routerImpl) Shutdown(ctx context.Context) error {
	return r.internal.Shutdown(ctx)
}

func (r *routerImpl) Group(path string, middlewares ...f.Middleware) f.HttpRouter {
	g := r.internal.Group(path, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := &httpContextImpl{
				internal: c,
				Context:  c.Request().Context(),
			}
			for _, middleware := range middlewares {
				if err := middleware(ctx); err != nil {
					return formatError(c, err, 0)
				}
			}
			return next(c)
		}
	})
	return &groupRouterImpl{
		internal:      g,
		tokenProvider: r.tokenProvider,
		ds:            r.ds,
	}
}

func (r *routerImpl) GET(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	nativeMiddlewares := make([]echo.MiddlewareFunc, 0)
	for _, m := range middlewares {
		if h.IsSameFunc(m, f.GzipMiddleware) {
			nativeMiddlewares = append(nativeMiddlewares, middleware.Gzip())
		}
	}
	r.internal.GET(path, wrapHandler(handler, r.ds, middlewares...), nativeMiddlewares...)
}

func (r *routerImpl) POST(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.POST(path, wrapHandler(handler, r.ds, middlewares...))
}

func (r *routerImpl) DELETE(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.DELETE(path, wrapHandler(handler, r.ds, middlewares...))
}

func (r *routerImpl) PUT(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.PUT(path, wrapHandler(handler, r.ds, middlewares...))
}

func (r *routerImpl) PATCH(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.PATCH(path, wrapHandler(handler, r.ds, middlewares...))
}

// ----

func (r *groupRouterImpl) GET(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	nativeMiddlewares := make([]echo.MiddlewareFunc, 0)
	for _, m := range middlewares {
		if h.IsSameFunc(m, f.GzipMiddleware) {
			nativeMiddlewares = append(nativeMiddlewares, middleware.Gzip())
		}
	}
	r.internal.GET(path, wrapHandler(handler, r.ds, middlewares...), nativeMiddlewares...)
}

func (r *groupRouterImpl) POST(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.POST(path, wrapHandler(handler, r.ds, middlewares...))
}

func (r *groupRouterImpl) DELETE(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.DELETE(path, wrapHandler(handler, r.ds, middlewares...))
}

func (r *groupRouterImpl) PUT(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.PUT(path, wrapHandler(handler, r.ds, middlewares...))
}

func (r *groupRouterImpl) PATCH(path string, handler func(c f.HttpContext) error, middlewares ...f.Middleware) {
	r.internal.PATCH(path, wrapHandler(handler, r.ds, middlewares...))
}

func wrapHandler(handler func(c f.HttpContext) error, dataSource f.DataSource, middlewares ...f.Middleware) echo.HandlerFunc {
	return func(c echo.Context) error {

		ctx := &httpContextImpl{
			internal: c,
			Context:  c.Request().Context(),
		}

		inTx := false

		defer func() {

			defaultCnx := ctx.Value(f.DefaultCnxKey{})
			tenantCnx := ctx.Value(f.TenantCnxKey{})

			if err := recover(); err != nil {
				tracerr.PrintSourceColor(tracerr.Wrap(err.(error)), 1)
				if inTx {
					if defaultCnx != nil {
						defaultCnx.(f.Connection).Rollback()
					}
					if tenantCnx != nil {
						tenantCnx.(f.Connection).Rollback()
					}
				}
				formatError(ctx.internal, err.(error), http.StatusInternalServerError)
			} else {
				if inTx {
					if defaultCnx != nil {
						defaultCnx.(f.Connection).Commit()
					}
					if tenantCnx != nil {
						tenantCnx.(f.Connection).Commit()
					}
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

		if dataSource != nil {
			defaultCnx := dataSource.DefaultConnection()
			if defaultCnx != nil {
				tx, err := defaultCnx.Tx(ctx)
				if err != nil {
					return err
				}
				defaultCnx = tx
				inTx = true
				ctx.Context = context.WithValue(ctx.Context, f.DefaultCnxKey{}, defaultCnx)
			}
			if ctx.TenantId() != "" {
				tenantCnx := dataSource.Connection(ctx.TenantId())
				if tenantCnx != nil {
					tx, err := tenantCnx.Tx(ctx)
					if err != nil {
						return err
					}
					tenantCnx = tx
					inTx = true
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

func (c *httpContextImpl) IdemPotencyKey() string {
	value := c.internal.Get(_idemPotencyKey)
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
	if customError, ok := err.(*errors.CustomError); ok {
		status = customError.Code
	}
	if status == 0 {
		status = http.StatusInternalServerError
	}
	errorMessage := err.Error()

	log.Error("http-error: %v -- %v", status, errorMessage)

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
