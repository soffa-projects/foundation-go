package f

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
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
	"github.com/soffa-projects/foundation-go/log"
	"github.com/soffa-projects/foundation-go/utils"
)

const _authKey = "auth"
const _authTokenKey = "authToken"
const _tenantIdKey = "tenantId"
const _envKey = "env"
const _connectionKey = "connection"

func NewEchoRouter(env Env, cfg *RouterConfig) Router {
	e := echo.New()
	e.Use(prettylogger.Logger)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogLevel: 2,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			tracerr.PrintSourceColor(tracerr.Wrap(err))
			return formatResponse(c, HttpResponse{
				Code: http.StatusInternalServerError, Data: err.Error()})
		},
	}))
	e.Use(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestID())

	e.StaticFS("/assets", cfg.AssetsFS)
	e.FileFS("/favicon.ico", "favicon.ico", cfg.FaviconFS)
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	//e.Use(session.MiddlewareWithConfig(session.Config{}))
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

	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set(_tenantIdKey, "")
			c.Set(_authKey, (*Authentication)(nil))
			c.Set(_envKey, env)
			authToken := ""
			authz := c.Request().Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				authToken = authz[len("bearer "):]
			}
			if env.TokenProvider != nil {
				token, err := env.TokenProvider.Verify(authToken)
				if err == nil {
					sub, _ := token.Subject()
					aud, _ := token.Audience()
					var permission string
					var email string
					_ = token.Get("permission", &permission)
					_ = token.Get("email", &email)
					//c.Set("authToken", authToken)
					auth := &Authentication{
						UserId:     sub,
						Audience:   aud,
						Permission: permission,
						Email:      email,
					}
					c.Set("auth", auth)
				}
			}
			c.Set(_authTokenKey, authToken)
			return next(c)
		}
	})

	return &routerImpl{
		internal:   e,
		production: env.Production,
		env:        env,
	}
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
	env      Env
}

func newRequestContext(c echo.Context, env Env) (Context, error) {
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

func (c *ctxImpl) Auth() *Authentication {
	value := c.internal.Get(_authKey)
	if value == nil {
		return nil
	}
	return value.(*Authentication)
}

func (c *ctxImpl) RealIP() string {
	return c.internal.RealIP()
}

func (c *ctxImpl) UserAgent() string {
	return c.internal.Request().UserAgent()
}

type routerImpl struct {
	internal   *echo.Echo
	production bool
	env        Env
}

func (r *routerImpl) GET(path string, handler HandlerInit) {
	r.internal.GET(path, r.wrap(handler))
}

func (r *routerImpl) POST(path string, handler HandlerInit) {
	r.internal.POST(path, r.wrap(handler))
}

func (r *routerImpl) DELETE(path string, handler HandlerInit) {
	r.internal.DELETE(path, r.wrap(handler))
}

func (r *routerImpl) PUT(path string, handler HandlerInit) {
	r.internal.PUT(path, r.wrap(handler))
}

func (r *routerImpl) PATCH(path string, handler HandlerInit) {
	r.internal.PATCH(path, r.wrap(handler))
}

func (r *routerImpl) Use(middleware Middleware) {
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

func (r *routerImpl) wrap(handlerInit HandlerInit) echo.HandlerFunc {

	handler := handlerInit(r.env)
	//isPublic := handler.Public

	return func(c echo.Context) error {
		// Authentication check

		rc, err := newRequestContext(c, r.env)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		var cnx Connection
		/*if rc.Auth() == nil && rc.AuthToken() == "" && !isPublic {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}*/

		// Handle panics and transactions
		defer func() {
			if rec := recover(); rec != nil {
				if !r.production {
					debug.PrintStack()
					//tracerr.PrintSourceColor(tracerr.Wrap(err))
				}
				if cnx != nil {
					if err := cnx.Rollback(); err != nil {
						log.Warn("db transaction rolled back: %v", err)
					}
				}
				c.Error(formatResponse(c, err))
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
			err := handler.Pre(rc)
			if err != nil {
				return formatResponse(c, err)
			}
		}

		if rc.Get(_connectionKey) != nil {
			cnx = rc.Get(_connectionKey).(Connection)
		} else {
			cnx = r.env.DS.Connection("default")
		}
		cnx, err = cnx.Tx(rc)
		if err != nil {
			return formatResponse(c, err)
		}

		result := handler.Handle(rc.WithValue(ConnectionKey{}, cnx))

		if result == nil {
			return formatResponse(c, HttpResponse{Code: http.StatusNoContent, Data: nil})
		}

		// Handle different response types
		switch v := result.(type) {
		case HttpResponse, error:
			return formatResponse(c, v)
		default:
			return c.JSON(http.StatusOK, result)
		}
	}
}

func formatResponse(c echo.Context, err any) error {

	switch err := err.(type) {
	case HttpResponse:
		break
	case error:
		return err
	}

	resp := err.(HttpResponse)

	if resp.Code >= 500 {
		return sendError(c, resp.Code, "err_technical", "err_unexpected_error")
	}

	if resp.Code >= 400 {
		return sendError(c, resp.Code, "err_functional", resp.Data)
	}

	if resp.Template != nil {
		return utils.RenderTempl(c, http.StatusOK, resp.Template)
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

func sendError(c echo.Context, status int, kind string, error any) error {
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

func (c *ctxImpl) Set(key string, value any) {
	c.internal.Set(key, value)
}

func (c *ctxImpl) Redirect(code int, url string) error {
	return c.internal.Redirect(code, url)
}

func (c *ctxImpl) Env() Env {
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
	}
}

func (c *ctxImpl) WithValue(key, value any) Context {
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

func (c *ctxImpl) File(data []byte, contentType string, filename string) HttpResponse {
	// Return the PDF data
	return HttpResponse{
		Code:        http.StatusOK,
		File:        true,
		Data:        data,
		ContentType: contentType,
		Filename:    filename,
	}
}

func (c *ctxImpl) Created(output any) HttpResponse {
	return HttpResponse{Code: http.StatusCreated, Data: output}
}

func (c *ctxImpl) Conflict(message string) HttpResponse {
	return HttpResponse{Code: http.StatusConflict, Data: message}
}

func (c *ctxImpl) NoContent() HttpResponse {
	return HttpResponse{Code: http.StatusNoContent}
}

func (c *ctxImpl) OK() HttpResponse {
	return HttpResponse{Code: http.StatusOK}
}

func (c *ctxImpl) BadRequest(message string) HttpResponse {
	return HttpResponse{Code: http.StatusBadRequest, Data: message}
}

func (c *ctxImpl) NotFound(message string) HttpResponse {
	return HttpResponse{Code: http.StatusNotFound, Data: message}
}

func (c *ctxImpl) Unauthorized(message string) HttpResponse {
	return HttpResponse{Code: http.StatusUnauthorized, Data: message}
}

func (c *ctxImpl) Forbidden(message string) HttpResponse {
	return HttpResponse{Code: http.StatusForbidden, Data: message}
}
