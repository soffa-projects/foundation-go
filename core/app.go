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
	PubSubProvider PubSubProvider
	SecretProvider SecretsProvider
	config         map[string]string
}

func Init(env ApplicationEnv, router Router, features []Feature) App {
	h.InitIdGenerator(0)

	features = checkFeatures(features...)
	for _, feature := range features {
		if feature.Init != nil {
			if err := feature.Init(&env); err != nil {
				log.Fatal("failed to initialize feature %s: %v", feature.Name, err)
			}
		}
	}
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
	if env.PubSubProvider != nil {
		if err := env.PubSubProvider.Init(); err != nil {
			log.Fatal("failed to initialize pubsub provider: %v", err)
		}
		Register(PubSubProviderKey, env.PubSubProvider)
	}
	if env.SecretProvider != nil {
		if err := env.SecretProvider.Init(); err != nil {
			log.Fatal("failed to initialize secret provider: %v", err)
		}
		Register(SecretProviderKey, env.SecretProvider)
	}
	router.Init(env)

	app := App{
		Env:    env,
		Router: router,
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
	Init       func(env *ApplicationEnv) error
	InitRoutes func(router Router)
}

func (app App) Start(port int) {
	defer func() {
		app.Shutdown(context.Background())
	}()
	log.Info("starting webserver")
	app.Router.Listen(port)
}

func (app App) Shutdown(ctx context.Context) {
	if err := app.Router.Shutdown(ctx); err != nil {
		log.Error("error shutting down server: %v", err)
	}
	if app.Env.SecretProvider != nil {
		if err := app.Env.SecretProvider.Close(); err != nil {
			log.Error("error shutting down secret provider: %v", err)
		}
	}
	log.Info("shutdown complete")
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
