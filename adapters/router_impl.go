package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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
const _connectionKey = "connection"

func NewEchoRouter(cfg *f.RouterConfig) f.Router {
	e := echo.New()
	e.Use(prettylogger.Logger)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogLevel: 2,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			tracerr.PrintSourceColor(tracerr.Wrap(err))
			return formatResponse(c, f.HttpResponse{
				Code: http.StatusInternalServerError, Data: err.Error()})
		},
	}))
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
						roles := ""
						var email string
						var tenantId string

						permissions := h.GetClaimValues(token, "permissions", "permission", "grant", "grants", "roles", "role")
						_ = token.Get("email", &email)
						_ = token.Get("roles", &roles)
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

type ctxImpl struct {
	context.Context
	internal echo.Context
	env      f.ApplicationEnv
}

func newRequestContext(c echo.Context, env f.ApplicationEnv) (f.Context, error) {
	return &ctxImpl{
		Context:  c.Request().Context(),
		internal: c,
		env:      env,
	}, nil
}

func (c *ctxImpl) Unwrap() context.Context {
	return c.internal.Request().Context()
}

func (c *ctxImpl) Get(key string) any {
	return c.internal.Get(key)
}

func (c *ctxImpl) AuthToken() string {
	value := c.internal.Get(_authTokenKey)
	if value == nil {
		return ""
	}
	return value.(string)
}

func (c *ctxImpl) Auth() *f.Authentication {
	value := c.internal.Get(_authKey)
	if value == nil {
		return nil
	}
	return value.(*f.Authentication)
}

func (c *ctxImpl) RealIP() string {
	return c.internal.RealIP()
}

func (c *ctxImpl) Host() string {
	return strings.ToLower(c.internal.Request().Host)
}

func (c *ctxImpl) UserAgent() string {
	return c.internal.Request().UserAgent()
}

type routerImpl struct {
	internal *echo.Echo
	env      f.ApplicationEnv
}

type groupRouterImpl struct {
	f.RouterGroup
	internal *echo.Group
	env      f.ApplicationEnv
}

func (r *routerImpl) Group(path string, middlewares ...f.Middleware) f.RouterGroup {
	g := r.internal.Group(path)
	if len(middlewares) > 0 {
		for _, middleware := range middlewares {
			g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					rc, err := newRequestContext(c, r.env)
					if err != nil {
						return err
					}
					if err := middleware(rc); err != nil {
						return formatResponse(c, err)
					}
					return next(c)
				}
			})
		}
	}
	return &groupRouterImpl{
		internal: g,
		env:      r.env,
	}
}

func (r *routerImpl) GET(path string, handler f.HandlerInit) {
	r.internal.GET(path, wrap(r.env, handler))
}

func (r *routerImpl) POST(path string, handler f.HandlerInit) {
	r.internal.POST(path, wrap(r.env, handler))
}

func (r *routerImpl) DELETE(path string, handler f.HandlerInit) {
	r.internal.DELETE(path, wrap(r.env, handler))
}

func (r *routerImpl) PUT(path string, handler f.HandlerInit) {
	r.internal.PUT(path, wrap(r.env, handler))
}

func (r *routerImpl) PATCH(path string, handler f.HandlerInit) {
	r.internal.PATCH(path, wrap(r.env, handler))
}

func (r *groupRouterImpl) GET(path string, handler f.HandlerInit) {
	r.internal.GET(path, wrap(r.env, handler))
}

func (r *groupRouterImpl) POST(path string, handler f.HandlerInit) {
	r.internal.POST(path, wrap(r.env, handler))
}

func (r *groupRouterImpl) DELETE(path string, handler f.HandlerInit) {
	r.internal.DELETE(path, wrap(r.env, handler))
}

func (r *groupRouterImpl) PUT(path string, handler f.HandlerInit) {
	r.internal.PUT(path, wrap(r.env, handler))
}

func (r *groupRouterImpl) PATCH(path string, handler f.HandlerInit) {
	r.internal.PATCH(path, wrap(r.env, handler))
}

func (r *routerImpl) Use(middleware f.Middleware) {
	r.internal.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rc, err := newRequestContext(c, r.env)
			if err != nil {
				return err
			}
			if err := middleware(rc); err != nil {
				return formatResponse(c, err)
			}
			return next(c)
		}
	})
}

func wrap(env f.ApplicationEnv, handlerInit f.HandlerInit) echo.HandlerFunc {

	handler := handlerInit(env)
	//isPublic := handler.Public

	return func(c echo.Context) error {
		// Authentication check

		rc, err := newRequestContext(c, env)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		var cnx f.Connection
		/*if rc.Auth() == nil && rc.AuthToken() == "" && !isPublic {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}*/

		// Handle panics and transactions
		defer func() {
			if rec := recover(); rec != nil {
				originalErr := rec.(error)
				if !env.Production {
					//debug.PrintStack()
					log.Error("panic: %v", originalErr)
					tracerr.PrintSourceColor(tracerr.Wrap(originalErr), 1)
				}
				if cnx != nil {
					if err := cnx.Rollback(); err != nil {
						log.Warn("db transaction rolled back: %v", err)
					}
				}
				c.Error(formatResponse(c, originalErr))
			} else {
				if cnx != nil {
					if err := cnx.Commit(); err != nil {
						c.Error(c.JSON(http.StatusInternalServerError, err))
					} else {
						log.Debug("db transaction committed")
					}
				}
			}
		}()

		if handler.Pre != nil {
			for _, pre := range handler.Pre {
				err := pre(rc)
				if err != nil {
					return formatResponse(c, err)
				}
			}
		}

		if handler.Authenticated && rc.Auth() == nil {
			return formatResponse(c, f.HttpResponse{Code: http.StatusUnauthorized, Data: "unauthorized"})
		}

		if handler.Permissions != nil {
			if !h.ContainsAnyString(handler.Permissions, rc.Auth().Permissions) {
				return formatResponse(c, f.HttpResponse{Code: http.StatusForbidden, Data: "forbidden"})
			}
		}

		var result any
		tenantId := rc.TenantId()

		if tenantId != "" {
			log.Info("active tenant is: %s", tenantId)
		}

		if env.DS != nil {
			if rc.Get(_connectionKey) != nil {
				cnx = rc.Get(_connectionKey).(f.Connection)
			} else {
				cnx = env.DS.Connection("default")
			}

			if cnx != nil {
				tx, err := cnx.Tx(rc)
				if err != nil {
					return formatResponse(c, err)
				}

				result = handler.Handle(rc.WithValue(f.TenantCnx{}, tx).WithValue(f.DefaultCnx{}, env.DS.Connection("default")))
			} else {
				result = handler.Handle(rc)
			}

		} else {
			result = handler.Handle(rc)
		}

		if result == nil {
			return formatResponse(c, f.HttpResponse{Code: http.StatusNoContent, Data: nil})
		}

		switch v := result.(type) {
		case f.HttpResponse, error, f.RedirectResponse:
			return formatResponse(c, v)
		default:
			return c.JSON(http.StatusOK, result)
		}
	}
}

func formatResponse(c echo.Context, err any) error {

	if err == nil {
		log.Error("unexpected empty error -- check that all interfaces have been implemented")
		return nil
	}

	if redirect, ok := err.(f.RedirectResponse); ok {
		return c.Redirect(redirect.Code, redirect.Url)
	}

	if _, ok := err.(f.HttpResponse); !ok {
		log.Error("unexpected error: %v", err)
		return c.JSON(http.StatusInternalServerError, err)
	}

	resp := err.(f.HttpResponse)

	if resp.Code == 204 {
		return c.NoContent(resp.Code)
	}

	if resp.Code >= 500 {
		log.Error("unexpected error: %v", err)
		return mapError(c, resp.Code, "err_technical", "err_unexpected_error")
	}

	if resp.Code >= 400 {
		log.Error("functional error: %v", err)
		return mapError(c, resp.Code, "err_functional", resp.Data)
	}

	if resp.Template != nil {
		return h.RenderTempl(c, http.StatusOK, resp.Template)
	}

	if resp.File {

		c.Response().Header().Set("Access-Control-Expose-Headers", "Content-Type,Content-Disposition, X-Filename")
		c.Response().Header().Set("Content-Type", resp.ContentType)
		c.Response().Header().Set("X-Filename", resp.Filename)
		c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", resp.Filename))

		return c.Blob(
			resp.Code,
			resp.ContentType,
			resp.Data.([]byte),
		)
	}

	return c.JSON(resp.Code, resp.Data)
}

func mapError(c echo.Context, status int, kind string, error any) error {
	return c.JSON(status, map[string]any{
		"requestId": c.Response().Header().Get(echo.HeaderXRequestID),
		"kind":      kind,
		"timestamp": time.Now().Format(time.RFC3339),
		"uri":       c.Request().URL.Path,
		"error":     error,
		"success":   false,
	})
}

func (c *ctxImpl) Param(value string) string {
	return c.internal.Param(value)
}

func (c *ctxImpl) QueryParam(value string) string {
	return c.internal.QueryParam(value)
}
func (c *ctxImpl) FormFile(field string) (io.ReadCloser, error) {
	file, err := c.internal.FormFile(field)
	if err != nil {
		return nil, err
	}
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	return src, nil
}

func (c *ctxImpl) Header(value string) string {
	return c.internal.Request().Header.Get(value)
}

func (c *ctxImpl) Set(key string, value any) {
	c.internal.Set(key, value)
}

func (c *ctxImpl) Env() f.ApplicationEnv {
	return c.env
}

func (c *ctxImpl) Bind(input any) {
	err := c.ShouldBind(input)
	if err != nil {
		panic(err)
	}
}

func (c *ctxImpl) SetTenant(tenantId string) {
	c.internal.Set(_tenantIdKey, tenantId)
	cnx := c.env.DS.Connection(tenantId)
	if cnx != nil {
		c.internal.Set(_connectionKey, cnx)
	} else {
		panic(fmt.Sprintf("tenant connexion %s not found", tenantId))
	}
}

func (c *ctxImpl) WithValue(key, value any) f.Context {
	return &ctxImpl{
		Context:  context.WithValue(c.Context, key, value),
		internal: c.internal,
		env:      c.env,
	}
}

func (c *ctxImpl) TenantId() string {
	value := c.internal.Get(_tenantIdKey)
	if value == nil {
		return ""
	}
	return value.(string)
}

func (c *ctxImpl) ShouldBind(input any) error {
	if err := c.internal.Bind(input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	binder := &echo.DefaultBinder{}
	if err := binder.BindHeaders(c.internal, input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func (c *ctxImpl) SetFlash(value string) error {
	return c.SetSession("flash", value, 15)
}

func (c *ctxImpl) UseFlash() (string, error) {
	sess, err := session.Get("session", c.internal)
	if err != nil {
		return "", err
	}
	value, ok := sess.Values["flash"]
	c.SetFlash("")
	if !ok {
		return "", nil
	}
	return value.(string), nil
}

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

func (c *ctxImpl) Created(output any) f.HttpResponse {
	return f.HttpResponse{Code: http.StatusCreated, Data: output}
}

func (c *ctxImpl) Conflict(message string) f.HttpResponse {
	return f.HttpResponse{Code: http.StatusConflict, Data: message}
}

func (c *ctxImpl) NoContent() f.HttpResponse {
	return f.HttpResponse{Code: http.StatusNoContent}
}

func (c *ctxImpl) OK() f.HttpResponse {
	return f.HttpResponse{Code: http.StatusOK}
}

func (c *ctxImpl) BadRequest(message string) f.HttpResponse {
	return f.HttpResponse{Code: http.StatusBadRequest, Data: message}
}

func (c *ctxImpl) NotFound(message string) f.HttpResponse {
	return f.HttpResponse{Code: http.StatusNotFound, Data: message}
}

func (c *ctxImpl) Unauthorized(message string) f.HttpResponse {
	return f.HttpResponse{Code: http.StatusUnauthorized, Data: message}
}

func (c *ctxImpl) Forbidden(message string) f.HttpResponse {
	return f.HttpResponse{Code: http.StatusForbidden, Data: message}
}

func (c *ctxImpl) Redirect(code int, url string) f.RedirectResponse {
	return f.RedirectResponse{Code: code, Url: url}
}
