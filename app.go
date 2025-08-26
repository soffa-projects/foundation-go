package f

import (
	"context"
	"io/fs"

	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

type NotificationService interface {
	Post(message string) error
}

type App struct {
	Router Router
	Env    Env
}

type RouterConfig struct {
	AllowOrigins  []string
	SessionSecret string
	AssetsFS      fs.FS
	FaviconFS     fs.FS
}

type LocalesConfig struct {
	LocaleFS fs.FS
	Locales  string
}

type Env struct {
	AppName       string
	BrandLogo     string
	Production    bool
	JwtSecret     string
	PublicURL     string
	I18n          I18n
	DS            DataSource
	TokenProvider TokenProvider
	EmailSender   EmailSender
	config        map[string]string
}

func Init(env Env, router Router, features []Feature) App {
	h.InitIdGenerator(0)
	app := App{
		Env:    env,
		Router: router,
	}
	for _, feature := range features {
		_ = feature(app)
	}
	return app
}

type FeatureSpec struct {
	//ApplyPatch func(ctx context.Context, db DB, patch int) (bool, error)
}

type Feature func(app App) error

func (app App) Start(port int) {
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

	log.Info("starting webserver")
	app.Router.Listen(port)
}

func (app App) Shutdown(ctx context.Context) {
	if err := app.Router.Shutdown(ctx); err != nil {
		log.Error("error shutting down server: %v", err)
	}
	log.Info("shutdown complete")
	//app.Scheduler.Stop()
}

// Init initializes the global generator
//func Init( ) {
//generator = NewShortIDGenerator(instanceId)
//}

func (env *Env) Config(key string) string {
	if val, ok := env.config[key]; ok {
		return val
	}
	return ""
}

func (env *Env) SetConfig(values map[string]string) {
	env.config = values
}
