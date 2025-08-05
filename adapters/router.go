package adapters

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/ztrue/tracerr"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	prettylogger "github.com/rdbell/echo-pretty-logger"
	log "github.com/sirupsen/logrus"
	core "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/utils"
)

const AuthTokenKey = "authToken"

func NewEchoRouter(env core.Env, cfg *core.RouterConfig) core.Router {
	e := echo.New()
	e.Use(prettylogger.Logger)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogLevel: 2,
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			tracerr.PrintSourceColor(tracerr.Wrap(err))
			return formatResponse(c, core.HttpResponse{
				Code: http.StatusInternalServerError, Data: err.Error()})
		},
	}))
	e.Use(middleware.RemoveTrailingSlash())
	e.Use(middleware.RequestID())
	e.StaticFS("/assets", cfg.AssetsFS)
	e.FileFS("/favicon.ico", "favicon.ico", cfg.FaviconFS)

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
			c.Set("tenantId", "")
			c.Set("tenant", nil)
			c.Set("auth", (*core.Authentication)(nil))
			c.Set("env", env)
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
					auth := &core.Authentication{
						UserId:     sub,
						Audience:   aud,
						Permission: permission,
						Email:      email,
					}
					c.Set("auth", auth)
				}
			}
			c.Set(AuthTokenKey, authToken)
			return next(c)
		}
	})

	if cfg != nil && cfg.Middlewares != nil {
		for _, middleware := range cfg.Middlewares {
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					rc, err := newRequestContext(c, env)
					if err != nil {
						return err
					}
					if err := middleware(rc); err != nil {
						tracerr.PrintSourceColor(tracerr.Wrap(err))
						return err
					}
					return next(c)
				}
			})
		}
	}

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
	log.Fatal(r.internal.Start(fmt.Sprintf(":%d", port)))
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
	core.Context
	internal echo.Context
	env      core.Env
}

func newRequestContext(c echo.Context, env core.Env) (core.Context, error) {
	return &ctxImpl{
		internal: c,
		env:      env,
	}, nil
}

func (c *ctxImpl) AppName() string {
	return c.internal.Get("appName").(string)
}

func (c *ctxImpl) AuthToken() string {
	value := c.internal.Get(AuthTokenKey)
	if value == nil {
		return ""
	}
	return value.(string)
}

func (c *ctxImpl) Auth() *core.Authentication {
	value := c.internal.Get("auth")
	if value == nil {
		return nil
	}
	return value.(*core.Authentication)
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
	env        core.Env
}

func (r *routerImpl) GET(path string, handler core.HandlerInit) {
	r.internal.GET(path, r.wrap(handler))
}

func (r *routerImpl) POST(path string, handler core.HandlerInit) {
	r.internal.POST(path, r.wrap(handler))
}

func (r *routerImpl) DELETE(path string, handler core.HandlerInit) {
	r.internal.DELETE(path, r.wrap(handler))
}

func (r *routerImpl) PUT(path string, handler core.HandlerInit) {
	r.internal.PUT(path, r.wrap(handler))
}

func (r *routerImpl) PATCH(path string, handler core.HandlerInit) {
	r.internal.PATCH(path, r.wrap(handler))
}

func (r *routerImpl) wrap(handlerInit core.HandlerInit) echo.HandlerFunc {

	handler := handlerInit(r.env)
	//isPublic := handler.Public

	return func(c echo.Context) error {
		// Authentication check

		rc, err := newRequestContext(c, r.env)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		/*if rc.Auth() == nil && rc.AuthToken() == "" && !isPublic {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}*/

		// Handle panics and transactions
		defer func() {
			if rec := recover(); rec != nil {

				if !r.production {
					// debug.PrintStack()
					tracerr.PrintSourceColor(tracerr.Wrap(err))
				}

				/*
					if rc.Tx() != nil {
						err = rc.Tx().Rollback()
						log.Errorf("db transaction rolled back: %v", err)
					}*/

				var err error
				switch v := rec.(type) {
				case error:
					err = v
				default:
					err = fmt.Errorf("%v", v)
				}

				log.Errorf("a technical error was raised: %v", err)
				c.Error(err)
			} else {
				/*if rc.Tx != nil {
					err = rc.Tx.Commit()
					if err != nil {
						log.Errorf("db transaction committed: %v", err)
					}
				}*/
			}
		}()

		if handler.PreAuthorize != nil {
			authorized, err := handler.PreAuthorize(rc)
			if err != nil {
				return err
			}
			if !authorized {
				return formatResponse(c, rc.Unauthorized("unauthorized"))
			}
		}

		result := handler.Handle(rc)

		if result == nil {
			return formatResponse(c, core.HttpResponse{Code: http.StatusNoContent, Data: nil})
		}

		// Handle different response types
		switch v := result.(type) {
		case core.HttpResponse:
			return formatResponse(c, v)
		case error:
			/*if rc.Tx != nil {
				_ = rc.Tx.Rollback()
			}*/
			log.Errorf("a technical error was raised: %v", v)
			return v
		default:
			return c.JSON(http.StatusOK, result)
		}
	}
}

func formatResponse(c echo.Context, resp core.HttpResponse) error {

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

func (c *ctxImpl) Env() core.Env {
	return c.env
}

func (c *ctxImpl) Bind(input any) error {
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

func (c *ctxImpl) File(data []byte, contentType string, filename string) core.HttpResponse {
	// Return the PDF data
	return core.HttpResponse{
		Code:        http.StatusOK,
		File:        true,
		Data:        data,
		ContentType: contentType,
		Filename:    filename,
	}
}

func (c *ctxImpl) Created(output any) core.HttpResponse {
	return core.HttpResponse{Code: http.StatusCreated, Data: output}
}

func (c *ctxImpl) Conflict(message string) core.HttpResponse {
	return core.HttpResponse{Code: http.StatusConflict, Data: message}
}

func (c *ctxImpl) NoContent() core.HttpResponse {
	return core.HttpResponse{Code: http.StatusNoContent}
}

func (c *ctxImpl) OK() core.HttpResponse {
	return core.HttpResponse{Code: http.StatusOK}
}

func (c *ctxImpl) BadRequest(message string) core.HttpResponse {
	return core.HttpResponse{Code: http.StatusBadRequest, Data: message}
}

func (c *ctxImpl) NotFound(message string) core.HttpResponse {
	return core.HttpResponse{Code: http.StatusNotFound, Data: message}
}

func (c *ctxImpl) Unauthorized(message string) core.HttpResponse {
	return core.HttpResponse{Code: http.StatusUnauthorized, Data: message}
}

func (c *ctxImpl) Forbidden(message string) core.HttpResponse {
	return core.HttpResponse{Code: http.StatusForbidden, Data: message}
}
