package f

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

type NotificationService interface {
	Post(message string) error
}

type App struct {
	Router Router
	Env    ApplicationEnv
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

type ApplicationEnv struct {
	AppName        string
	Version        string
	BrandLogo      string
	Production     bool
	JwtSecret      string
	PublicURL      string
	I18n           I18n
	DS             DataSource
	TenantProvider TenantProvider
	TokenProvider  TokenProvider
	EmailSender    EmailSender
	config         map[string]string
}

func Init(env ApplicationEnv, router Router, features []Feature) App {
	h.InitIdGenerator(0)
	app := App{
		Env:    env,
		Router: router,
	}
	features = checkFeatures(features...)
	if env.TenantProvider != nil {
		if err := env.TenantProvider.Init(features); err != nil {
			log.Fatal("failed to initialize tenant provider: %v", err)
		}
	}
	if env.DS != nil {
		if err := env.DS.Init(env, features); err != nil {
			log.Fatal("failed to initialize data source: %v", err)
		}
	}
	for _, feature := range features {
		feature.InitRoutes(app.Router)
	}
	return app
}

type FeatureSpec struct {
	//ApplyPatch func(ctx context.Context, db DB, patch int) (bool, error)
}

type Feature struct {
	Name       string
	FS         fs.FS
	DependsOn  []Feature
	InitRoutes func(router Router)
}

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

func (env *ApplicationEnv) Config(key string) string {
	if val, ok := env.config[key]; ok {
		return val
	}
	return ""
}

func (env *ApplicationEnv) SetConfig(values map[string]string) {
	env.config = values
}

type TenantInput struct {
	Tenant string `param:"tenant" header:"X-TenantId" json:"-" validate:"required"`
}

func checkFeatures(features ...Feature) []Feature {
	featureMap := make(map[string]bool, len(features))
	loadedFeatures := []Feature{}
	for _, f := range features {
		if f.Name == "" {
			log.Fatal("feature name is required")
		}
		if _, ok := featureMap[f.Name]; ok {
			log.Fatal("feature name %s is already registered", f.Name)
		}
		featureMap[f.Name] = true
		loadedFeatures = append(loadedFeatures, f)
	}
	// make sure dependencies are loaded
	for _, f := range loadedFeatures {
		for _, dep := range f.DependsOn {
			if _, ok := featureMap[dep.Name]; !ok {
				loadedFeatures = append(loadedFeatures, dep)
				featureMap[dep.Name] = true
			}
		}
	}
	return orderFeatures(loadedFeatures)
}

func orderFeatures(features []Feature) []Feature {
	// Map features by name for lookup
	featureMap := make(map[string]Feature, len(features))
	for _, f := range features {
		featureMap[f.Name] = f
	}

	visited := make(map[string]bool)
	temp := make(map[string]bool) // for cycle detection
	var ordered []Feature

	var visit func(string) error

	visit = func(name string) error {
		if visited[name] {
			return nil
		}
		if temp[name] {
			log.Fatal("cyclic dependency detected: %s", name)
		}

		temp[name] = true
		feat, ok := featureMap[name]
		if !ok {
			return fmt.Errorf("unknown dependency: %s", name)
		}

		for _, dep := range feat.DependsOn {
			if err := visit(dep.Name); err != nil {
				return err
			}
		}

		temp[name] = false
		visited[name] = true
		ordered = append(ordered, feat)
		return nil
	}

	// Visit all features
	for _, f := range features {
		if !visited[f.Name] {
			if err := visit(f.Name); err != nil {
				log.Fatal("error visiting feature: %s -- %v", f.Name, err)
				return nil
			}
		}
	}

	return ordered
}
