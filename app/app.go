package app

import (
	"context"
	"fmt"
	"time"

	adapters "github.com/soffa-projects/foundation-go/adapters"
	f "github.com/soffa-projects/foundation-go/core"
	"github.com/soffa-projects/foundation-go/h"
	"github.com/soffa-projects/foundation-go/log"
	"github.com/thoas/go-funk"
)

type IdempotencyProvider struct {
	ttl string
}

type builderConfig struct {
	appName             string
	appVersion          string
	envName             string
	sessionSecret       string
	publicURL           string
	i18n                *f.LocalesConfig
	emailSender         string
	pubSubProvider      string
	cacheProvider       string
	secretProvider      f.SecretsProvider
	errorReporter       string
	queueProvider       string
	tokenProvider       *f.JwtConfig
	dsConfig            []f.DataSourceConfig
	tenantProvider      string
	config              any
	routerConfig        f.RouterConfig
	instanceId          string
	idempotencyProvider *IdempotencyProvider
	authProvider        f.AuthProvider
	logLevel            string
}

type AppBuilder struct {
	config builderConfig
}

type appImpl struct {
	f.App
	router     f.Router
	instanceId string
}

func (app *appImpl) Start(port int) error {
	defer func() {
		app.Shutdown(context.Background())
	}()

	log.Info("starting webserver...")
	if err := app.router.Listen(port); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}
	return nil
}

func (app *appImpl) Router() f.Router {
	return app.router
}

func (app *appImpl) Shutdown(ctx context.Context) {
	if err := app.router.Shutdown(ctx); err != nil {
		log.Error("error shutting down server: %v", err)
	}
	/*
		TODO: implement shutdown hooks for all providers
		if app.env.secretProvider != nil {
			if err := app.env.secretProvider.Close(); err != nil {
				log.Error("error shutting down secret provider: %v", err)
			}
		}*/
	log.Info("shutdown complete")
}

func New(name string, version string, envName string) AppBuilder {
	return AppBuilder{
		config: builderConfig{
			appName:    name,
			appVersion: version,
			envName:    envName,
			config:     make(map[any]string),
			instanceId: h.RandomString(32),
		},
	}
}

// Init initializes the application with the given features and returns an error if initialization fails.
// For cases where you want initialization failures to panic, use MustInit() instead.
func (app AppBuilder) Init(features []f.Feature) (f.App, error) {

	log.Init(app.config.logLevel)

	log.Info("initializing application...")

	h.InitIdGenerator(0)

	production := h.IsProduction(app.config.envName)

	cfg := app.config
	appInfo := f.AppInfo{
		Name:      cfg.appName,
		Version:   cfg.appVersion,
		PublicURL: cfg.publicURL,
	}
	instanceId := cfg.instanceId

	/*env := applicationEnvImpl{
		appName:    app.config.appName,
		appVersion: app.config.appVersion,
		envName:    cfg.envName,
		production: h.IsProduction(cfg.envName),
		publicURL:  cfg.publicURL,
	}*/

	orderedFeatures, err := checkFeatures(features...)
	if err != nil {
		return nil, err
	}
	features = orderedFeatures

	initContext := f.InitContext{
		InstanceId: instanceId,
		Config:     cfg.config,
	}

	log.Info("preloading features...")
	for _, feature := range features {
		if feature.OnInit == nil {
			return nil, fmt.Errorf("feature %s has no create function", feature.Name)
		}
		// Preload singletons here
		if feature.BeforeInit != nil {
			feature.BeforeInit(initContext)
		}
	}

	log.Info("initializing infrastructure...")

	var tenantProvider f.TenantProvider
	var tokenProvider f.TokenProvider
	var dataSource f.DataSource

	if !funk.IsEmpty(cfg.i18n) {
		adapter, err := adapters.NewLocalizer(cfg.i18n.LocaleFS, cfg.i18n.Locales)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize localizer: %v", err)
		}
		f.Provide(adapter)
	}
	if !funk.IsEmpty(cfg.tenantProvider) {
		adapter, err := adapters.NewTenantProvider(cfg.tenantProvider)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize tenant provider: %v", err)
		}
		tenantProvider = adapter
		f.Provide(tenantProvider)
	} else {
		adapter := f.Lookup[f.TenantProvider]()
		if adapter != nil {
			tenantProvider = *adapter
		}
	}
	if cfg.dsConfig != nil {
		adapter := adapters.NewMultiTenantDS(cfg.dsConfig...)
		if tenantProvider != nil {
			adapter.UseTenantProvider(tenantProvider)
		}
		if err := adapter.Init(features); err != nil {
			return nil, fmt.Errorf("[000] failed to initialize wMultiTenantDS: %v", err)
		}
		// ds = adapter
		dataSource = adapter
		f.Provide[f.DataSource](adapter)
		f.Provide(adapters.NewEntityManagerImpl(adapter))
	}
	if !funk.IsEmpty(cfg.emailSender) {
		adapter, err := adapters.NewEmailSender(cfg.appName, cfg.emailSender)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize email sender: %v", err)
		}
		f.Provide(adapter)
	}
	if !funk.IsEmpty(cfg.pubSubProvider) {
		adapter, err := adapters.NewPubSubProvider(cfg.pubSubProvider)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize pubsub provider: %v", err)
		}
		if err := adapter.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize pubsub provider: %v", err)
		}
		f.Provide(adapter)
	}
	if !funk.IsEmpty(cfg.cacheProvider) {
		adapter, err := adapters.NewCacheProvider(cfg.cacheProvider)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cache provider: %v", err)
		}
		if err := adapter.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize cache provider: %v", err)
		}
		f.Provide(adapter)

		idempotencyStore := adapters.NewIdempotencyStore(adapter, 1*time.Hour)
		f.Provide(idempotencyStore)
	}
	if !funk.IsEmpty(cfg.queueProvider) {
		adapter, err := adapters.NewAsynqQueueProvider(cfg.queueProvider)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize queue provider: %v", err)
		}
		f.Provide(adapter)
	}
	if cfg.secretProvider != nil {
		if err := cfg.secretProvider.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize secret provider: %v", err)
		}
		f.Provide(cfg.secretProvider)
		log.Info("secret provider initialized and registered")
	} else {
		log.Debug("no secret provider provided")
	}
	if !funk.IsEmpty(cfg.errorReporter) {
		adapter := adapters.NewSentryErrorReporter(cfg.errorReporter, cfg.envName)
		f.Provide(adapter)
	}
	if !funk.IsEmpty(cfg.tokenProvider) {
		adapter, err := adapters.NewTokenProvider(f.JwtConfig{
			Issuer:           cfg.appName,
			JwkPrivateBase64: cfg.tokenProvider.JwkPrivateBase64,
			JwkPublicBase64:  cfg.tokenProvider.JwkPublicBase64,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize token provider: %v", err)
		}
		tokenProvider = adapter
		f.Provide(tokenProvider)
	}

	f.Provide(adapters.NewCsrfTokenProvider())
	f.Provide(appInfo)

	router := adapters.NewEchoRouter(adapters.EchoRouterConfig{
		Debug:          !production,
		PublicFS:       cfg.routerConfig.PublicFS,
		SessionSecret:  cfg.routerConfig.SessionSecret,
		AllowOrigins:   cfg.routerConfig.AllowOrigins,
		SentryDSN:      cfg.routerConfig.SentryDSN,
		Env:            cfg.envName,
		TokenProvider:  tokenProvider,
		TenantProvider: tenantProvider,
		DataSource:     dataSource,
		AuthProvider:   cfg.authProvider,
	})

	router.Init()

	mcp := adapters.NewMCPServer(f.MCPServerConfig{
		ToolsCapabilities:   true,
		PromptsCapabilities: false,
	})

	mcp.Init(appInfo)

	initContext.Router = router
	initContext.MCP = mcp

	log.Info("initializing features...")
	for _, feature := range features {
		feature.OnInit(initContext)
		log.Info("feature %s initialized", feature.Name)
	}

	if !mcp.IsEmpty() {
		router.MCP("/mcp", mcp.HttpHandler())
	}

	return &appImpl{
		router:     router,
		instanceId: instanceId,
	}, nil
}

// MustInit is a convenience wrapper around Init that panics on error.
// Use this only in main() or initialization code where panic is acceptable.
func (app AppBuilder) MustInit(features []f.Feature) f.App {
	result, err := app.Init(features)
	if err != nil {
		panic(err)
	}
	return result
}

func (app *appImpl) InstanceId() string {
	return app.instanceId
}

func (app AppBuilder) WithInstanceId(id string) AppBuilder {
	app.config.instanceId = id
	return app
}

func (app AppBuilder) WithPublicURL(url string) AppBuilder {
	app.config.publicURL = url
	return app
}

func (app AppBuilder) WithAuthProvider(authProvider f.AuthProvider) AppBuilder {
	app.config.authProvider = authProvider
	return app
}

func (app AppBuilder) WithI18n(cfg f.LocalesConfig) AppBuilder {
	app.config.i18n = &cfg
	return app
}

func (app AppBuilder) WithEmailSender(provider string) AppBuilder {
	app.config.emailSender = provider
	return app
}

func (app AppBuilder) WithPubSubProvider(provider string) AppBuilder {
	app.config.pubSubProvider = provider
	return app
}

func (app AppBuilder) WithCacheProvider(provider string) AppBuilder {
	app.config.cacheProvider = provider
	return app
}

func (app AppBuilder) WithIdempotencyProvider(ttl string) AppBuilder {
	app.config.idempotencyProvider = &IdempotencyProvider{ttl: ttl}
	return app
}

func (app AppBuilder) WithSecretProvider(provider f.SecretsProvider) AppBuilder {
	app.config.secretProvider = provider
	return app
}

func (app AppBuilder) WithQueueProvider(provider string) AppBuilder {
	app.config.queueProvider = provider
	return app
}

func (app AppBuilder) WithLogLevel(level string) AppBuilder {
	app.config.logLevel = level
	return app
}

func (app AppBuilder) WithErrorReporter(provider string) AppBuilder {
	app.config.errorReporter = provider
	return app
}

func (app AppBuilder) WithTokenProvider(config f.JwtConfig) AppBuilder {
	app.config.tokenProvider = &config
	return app
}

func (app AppBuilder) WithConfig(config any) AppBuilder {
	app.config.config = config
	return app
}

func (app AppBuilder) WithDataSource(config ...f.DataSourceConfig) AppBuilder {
	if len(config) == 0 {
		app.config.dsConfig = []f.DataSourceConfig{}
	} else {
		app.config.dsConfig = config
	}
	return app
}

func (app AppBuilder) WithTenantProvider(prpovider string) AppBuilder {
	app.config.tenantProvider = prpovider
	return app
}

func (app AppBuilder) WithSessionSecret(secret string) AppBuilder {
	app.config.sessionSecret = secret
	return app
}

func (app AppBuilder) WithRouterConfig(cfg f.RouterConfig) AppBuilder {
	app.config.routerConfig = cfg
	return app
}

func checkFeatures(features ...f.Feature) ([]f.Feature, error) {
	featureMap := make(map[string]bool, len(features))
	loadedFeatures := []f.Feature{}
	for _, f := range features {
		if f.Name == "" {
			return nil, fmt.Errorf("feature name is required")
		}
		if _, ok := featureMap[f.Name]; ok {
			return nil, fmt.Errorf("feature name %s is already registered", f.Name)
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

func orderFeatures(features []f.Feature) ([]f.Feature, error) {
	// Map features by name for quick lookup
	featureMap := make(map[string]f.Feature, len(features))
	indegree := make(map[string]int, len(features)) // number of dependencies
	graph := make(map[string][]string)              // adjacency list

	// Initialize maps
	for _, f := range features {
		featureMap[f.Name] = f
		if _, ok := indegree[f.Name]; !ok {
			indegree[f.Name] = 0
		}
	}

	// Build graph and indegree count
	for _, f := range features {
		for _, dep := range f.DependsOn {
			graph[dep.Name] = append(graph[dep.Name], f.Name)
			indegree[f.Name]++
		}
	}

	// Queue of features with no dependencies
	queue := make([]string, 0)
	for name, deg := range indegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}

	// Perform topological sort
	var ordered []f.Feature
	for len(queue) > 0 {
		// Pop from queue
		name := queue[0]
		queue = queue[1:]

		feat, ok := featureMap[name]
		if !ok {
			return nil, fmt.Errorf("unknown feature: %s", name)
		}
		ordered = append(ordered, feat)

		// Decrease indegree of dependents
		for _, neigh := range graph[name] {
			indegree[neigh]--
			if indegree[neigh] == 0 {
				queue = append(queue, neigh)
			}
		}
	}

	// Check for cycles
	if len(ordered) != len(features) {
		return nil, fmt.Errorf("cyclic dependency detected - %v / %v", len(ordered), len(features))
	}

	return ordered, nil
}
