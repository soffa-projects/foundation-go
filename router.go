package micro

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/ztrue/tracerr"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	prettylogger "github.com/rdbell/echo-pretty-logger"
	log "github.com/sirupsen/logrus"
)

type Router struct {
	internal   *echo.Echo
	production bool
}

type Authentication struct {
	//TenantId   string
	UserId     string
	Audience   []string
	Permission string
	Email      string
}

func NewEchoRouter(env *Env, cfg *RouterConfig) *Router {
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
			c.Set("auth", (*Authentication)(nil))
			c.Set("env", env)
			authToken := ""
			authz := c.Request().Header.Get("Authorization")
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				authToken = authz[len("bearer "):]
			}
			if env.JwtProvider != nil {
				token, err := env.JwtProvider.Verify(authToken)
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
			c.Set("authToken", authToken)
			return next(c)
		}
	})

	if cfg != nil && cfg.Middlewares != nil {
		for _, middleware := range cfg.Middlewares {
			e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					rc, err := newCtx(c)
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

	return &Router{
		internal:   e,
		production: env.Production,
	}
}

func (r Router) Handler() http.Handler {
	return r.internal
}

func (r Router) Listen(port int) {
	if port == 0 {
		port = 8080
	}
	log.Fatal(r.internal.Start(fmt.Sprintf(":%d", port)))
}

func (r Router) Shutdown(ctx context.Context) error {
	return r.internal.Shutdown(ctx)
}

type RouteOpt struct {
	Public bool
	//Tenant bool
}

type Ctx struct {
	context.Context
	//echo.Context
	internal echo.Context
	//Context  context.Context
	//UserID    string
	Auth      *Authentication
	EM        *EntityManager
	RealIP    string
	UserAgent string
	Env       *Env
	Tx        DB
	AuthToken string
	TenantId  string
	Tenant    any
}

func newCtx(c echo.Context) (*Ctx, error) {
	//value := c.Get("userId")
	env := c.Get("env").(*Env)
	//userId, _ := value.(string)
	authToken := c.Get("authToken").(string)
	tenantId := c.Get("tenantId").(string)
	tenant := c.Get("tenant")
	auth := c.Get("auth").(*Authentication)

	// Setup context and database
	ctx := context.Background()
	var tx DB
	//var err error

	if tenantId != "" {
		ctx = context.WithValue(ctx, TenantID{}, tenantId)
	}
	/*if env.DB != nil {
		tx, err = env.DB.Tx(ctx, &sql.TxOptions{
			ReadOnly:  false,
			Isolation: sql.LevelDefault,
		})
		if err != nil {
			return nil, err
		}
	}*/
	userAgent := c.Request().UserAgent()
	return &Ctx{
		//Context:   c,
		Auth:      auth,
		EM:        &env.EM,
		internal:  c,
		Context:   context.WithValue(ctx, DBIKey{}, env.EM.DefaultTenant),
		RealIP:    c.RealIP(),
		UserAgent: userAgent,
		Tx:        tx,
		Env:       env,
		AuthToken: authToken,
		TenantId:  tenantId,
		Tenant:    tenant,
	}, nil
}

func (r *Ctx) UseTenant() context.Context {
	return context.WithValue(r.Context, TenantID{}, r.TenantId)
}

func (r Router) GET(path string, handler any, opts ...RouteOpt) {
	r.internal.GET(path, r.wrap(handler, opts))
}

func (r Router) POST(path string, handler any, opts ...RouteOpt) {
	r.internal.POST(path, r.wrap(handler, opts))
}

func (r Router) DELETE(path string, handler any, opts ...RouteOpt) {
	r.internal.DELETE(path, r.wrap(handler, opts))
}

func (r Router) PUT(path string, handler any, opts ...RouteOpt) {
	r.internal.PUT(path, r.wrap(handler, opts))
}

func (r Router) PATCH(path string, handler any, opts ...RouteOpt) {
	r.internal.PATCH(path, r.wrap(handler, opts))
}

func (r Router) wrap(h any, opts []RouteOpt) echo.HandlerFunc {
	// Check if route is public
	isPublic := false
	//hasTenant := false
	for _, opt := range opts {
		if opt.Public {
			isPublic = true
		}
		/*if opt.Tenant {
			hasTenant = true
		}*/
	}

	// Validate handler signature once when the route is registered
	handlerType := reflect.TypeOf(h)
	if handlerType.Kind() != reflect.Func {
		panic("router: handler must be a function")
	}

	// Validate input parameters
	if handlerType.NumIn() < 1 || handlerType.NumIn() > 2 {
		panic("router: handler must take 1 or 2 parameters")
	}

	// First parameter must be *RequestContext
	if handlerType.In(0) != reflect.TypeOf(&Ctx{}) {
		panic("router: first parameter must be *RequestContext")
	}

	// Check for optional second parameter
	hasInput := handlerType.NumIn() == 2
	var inputType reflect.Type
	if hasInput {
		log.Debugf("router handler has an input arg")
		inputType = handlerType.In(1)
		// Second parameter must be a struct (not pointer)
		if inputType.Kind() != reflect.Struct {
			panic("router: second parameter must be a struct (passed by value)")
		}
	}

	// Validate return type (must return exactly one value)
	if handlerType.NumOut() != 1 {
		panic("router: handler must return exactly one value")
	}

	return func(c echo.Context) error {
		// Authentication check

		rc, err := newCtx(c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}

		if rc.Auth == nil && rc.AuthToken == "" && !isPublic {
			return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
		}

		// Handle panics and transactions
		defer func() {
			if rec := recover(); rec != nil {

				if !r.production {
					// debug.PrintStack()
					tracerr.PrintSourceColor(tracerr.Wrap(err))
				}

				if rc.Tx != nil {
					err = rc.Tx.Rollback()
					log.Errorf("db transaction rolled back: %v", err)
				}

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
				if rc.Tx != nil {
					err = rc.Tx.Commit()
					if err != nil {
						log.Errorf("db transaction committed: %v", err)
					}
				}
			}
		}()

		// Prepare arguments for handler call
		args := []reflect.Value{reflect.ValueOf(rc)}

		if hasInput {
			// Create new struct instance
			inputValue := reflect.New(inputType)

			// Bind to the pointer (echo's Bind expects a pointer)
			if err := c.Bind(inputValue.Interface()); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid input format")
			}

			validate := validator.New()
			if err := validate.Struct(inputValue.Interface()); err != nil {
				log.Errorf("invalid input format: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, err.Error())
			}

			// Convert back to value and add to args
			args = append(args, inputValue.Elem())
		}

		// Call the handler
		results := reflect.ValueOf(h).Call(args)
		if len(results) == 0 {
			return nil
		}

		result := results[0].Interface()

		if result == nil {
			return formatResponse(c, HttpResponse{Code: http.StatusNoContent, Data: nil})
		}

		// Handle different response types
		switch v := result.(type) {
		case HttpResponse:
			return formatResponse(c, v)
		case error:
			if rc.Tx != nil {
				_ = rc.Tx.Rollback()
			}
			log.Errorf("a technical error was raised: %v", v)
			return v
		default:
			return c.JSON(http.StatusOK, result)
		}
	}
}

func formatResponse(c echo.Context, resp HttpResponse) error {

	if resp.Code >= 500 {
		return sendError(c, resp.Code, "err_technical", "err_unexpected_error")
	}

	if resp.Code >= 400 {
		return sendError(c, resp.Code, "err_functional", resp.Data)
	}

	if resp.Template != nil {
		return RenderTempl(c, http.StatusOK, resp.Template)
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

func (c *Ctx) Param(value string) string {
	return c.internal.Param(value)
}

func (c *Ctx) Set(key string, value any) {
	c.internal.Set(key, value)
}

func (c *Ctx) Redirect(code int, url string) error {
	return c.internal.Redirect(code, url)
}

func (c *Ctx) Bind(input any) {
	if err := c.internal.Bind(input); err != nil {
		panic(echo.NewHTTPError(http.StatusBadRequest, err))
	}
	validate := validator.New()
	if err := validate.Struct(input); err != nil {
		panic(echo.NewHTTPError(http.StatusBadRequest, err.Error()))
	}
}

func (c *Ctx) SetFlash(value string) error {
	return c.SetSession("flash", value, 15)
}

func (c *Ctx) UseFlash() (string, error) {
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

func (c *Ctx) SetSession(key string, value string, maxAge int) error {
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

func (c *Ctx) File(data []byte, contentType string, filename string) HttpResponse {
	// Return the PDF data
	return HttpResponse{
		Code:        http.StatusOK,
		File:        true,
		Data:        data,
		ContentType: contentType,
		Filename:    filename,
	}
}

func (c *Ctx) Created(output any) HttpResponse {
	return HttpResponse{Code: http.StatusCreated, Data: output}
}

func (c *Ctx) Conflict(message any) HttpResponse {
	return HttpResponse{Code: http.StatusConflict, Data: message}
}

func (c *Ctx) NoContent() HttpResponse {
	return HttpResponse{Code: http.StatusNoContent}
}

func (c *Ctx) OK() HttpResponse {
	return HttpResponse{Code: http.StatusOK}
}

func (c *Ctx) BadRequest(message any) any {
	return HttpResponse{Code: http.StatusBadRequest, Data: message}
}

func (c *Ctx) NotFound(message string) error {
	return HttpResponse{Code: http.StatusNotFound, Data: message}
}

func (c *Ctx) Unauthorized(message string) any {
	return HttpResponse{Code: http.StatusUnauthorized, Data: message}
}

func (c *Ctx) Forbidden(message string) any {
	return HttpResponse{Code: http.StatusForbidden, Data: message}
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
