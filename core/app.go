package f

import (
	"context"

	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

type App struct {
	Router Router
	Env    ApplicationEnv
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

func (env *ApplicationEnv) Config(key string) string {
	if val, ok := env.config[key]; ok {
		return val
	}
	return ""
}

func (env *ApplicationEnv) SetConfig(values map[string]string) {
	env.config = values
}
