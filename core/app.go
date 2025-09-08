package f

import (
	"context"

	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
)

type App struct {
	Env    ApplicationEnv
	Router Router
}

type ApplicationEnv struct {
	AppName           string
	AppVersion        string
	BrandLogo         string
	Production        bool
	JwtSecret         string
	PublicURL         string
	I18n              I18n
	DS                DataSource
	TenantProvider    TenantProvider
	TokenProvider     TokenProvider
	EmailSender       EmailSender
	PubSubProvider    PubSubProvider
	CacheProvider     CacheProvider
	SecretProvider    SecretsProvider
	CsrfTokenProvider CsrfTokenProvider
	config            map[string]string
	ErrorReporter     ErrorReporter
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

	env.CsrfTokenProvider = NewCsrfTokenProvider()
	router.Init(env)

	mcp := newMCPServer(MCPServerConfig{
		ToolsCapabilities:   true,
		PromptsCapabilities: false,
	})
	mcp.Init(env)

	for _, feature := range features {
		if feature.InitRoutes != nil {
			feature.InitRoutes(router)
		}
		if feature.Operations != nil {
			for _, operation := range feature.Operations {
				router.AddOperation(operation)
				mcp.AddOperation(operation)
			}
		}
	}
	router.MCP("/mcp", mcp.HttpHandler())
	return App{
		Env:    env,
		Router: router,
	}
}

func (app App) Start(port int) {
	defer func() {
		app.Shutdown(context.Background())
	}()

	log.Info("starting webserver...")
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
