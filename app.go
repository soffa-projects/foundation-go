package micro

import (
	"context"
	"io/fs"
)

type NotificationService interface {
	Post(message string) error
}

type Env struct {
	AppName   string
	EM        EntityManager
	PublicURL string
	//TenantLoader TenantLoader
	JwtProvider JwtProvider
	EmailSender EmailSender
	L           *I18n
	Config      map[string]string
	Production  bool
}

type App struct {
	R   *Router
	Env *Env
	//	Scheduler Scheduler
	Context context.Context
}

type RouterConfig struct {
	AllowOrigins  []string
	SessionSecret string
	Middlewares   []func(c *Ctx) error
	AssetsFS      fs.FS
	FaviconFS     fs.FS
}

type DatabaseConfig struct {
	DatabaseURL  string
	TenatLoader  TenantLoader
	MigrationsFS fs.FS
}

type LocalesConfig struct {
	LocaleFS fs.FS
	Locales  string
}

type Config struct {
	AppName       string
	Production    bool
	DB            *DatabaseConfig
	JwtSecret     string
	EmailProvider string
	PublicURL     string
	Locales       *LocalesConfig
	//S3Config           *S3Config
	Router *RouterConfig
	Jwt    *JwtConfig
	Config map[string]string
}

func Init(cfg Config, features []Feature) *App {

	env := &Env{
		AppName:    cfg.AppName,
		PublicURL:  cfg.PublicURL,
		Config:     cfg.Config,
		Production: cfg.Production,
	}
	if cfg.DB != nil {
		env.EM = createEntityManager(*cfg.DB)
	}
	if cfg.Locales != nil {
		env.L = createLocalizer(cfg.Locales.LocaleFS, cfg.Locales.Locales)
	}
	if cfg.Jwt != nil {
		env.JwtProvider = NewJwtProvider(*cfg.Jwt)
	}
	if cfg.EmailProvider != "" {
		emailSender, err := ConfigureEmailProvider(cfg.EmailProvider)
		if err != nil {
			LogFatal("failed to configure email provider: %v", err)
		}
		env.EmailSender = emailSender
	}
	/*if cfg.S3Config != nil {
		env.S3Client = NewS3Client(cfg.S3Config)
	}*/
	rootCtx := context.Background()
	InitIdGenerator(0)

	app := &App{
		Env: env,
		R:   NewEchoRouter(env, cfg.Router),
		//Scheduler: NewDefaultSchedulerImpl(env),
		Context: rootCtx,
	}

	for _, feature := range features {
		_ = feature(app)
	}

	return app
}

type FeatureSpec struct {
	//ApplyPatch func(ctx context.Context, db DB, patch int) (bool, error)
}

type Feature func(app *App) *FeatureSpec

func (app *App) Run(port int) {
	// start scheduler

	/*
		rootCtx, cancel := context.WithCancel(app.Context)
		// Set up graceful shutdown
		go func() {
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt)
			<-quit
			LogInfo("shutdown notice received ...")
			ctx, shutdownCancel := context.WithTimeout(rootCtx, 5*time.Second)
			defer shutdownCancel()
			app.Shutdown(ctx)
			cancel() // Cancel the Redis listener when the server is shutting down
		}()

		defer cancel()


			go func() {
				LogInfo("starting cron scheduler")
				//app.Scheduler.Start()
			}()
	*/

	LogInfo("starting webserver")
	app.R.Listen(port)
	//app.R.Logger.Fatal(app.R.Start(fmt.Sprintf(":%d", port)))
}

func (app *App) Shutdown(ctx context.Context) {
	if err := app.R.Shutdown(ctx); err != nil {
		LogError("error shutting down server: %v", err)
	}
	LogInfo("shutdown complete")
	//app.Scheduler.Stop()
}

// Init initializes the global generator
//func Init( ) {
//generator = NewShortIDGenerator(instanceId)
//}
