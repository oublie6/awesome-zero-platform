package bootstrap

import (
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/config"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/handler"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/foundation/httpmiddleware"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	restrouter "github.com/zeromicro/go-zero/rest/router"
)

type App struct {
	Config config.Config
	server *rest.Server
}

func New(configFile string) (*App, error) {
	var cfg config.Config
	if err := conf.Load(configFile, &cfg); err != nil {
		return nil, fmt.Errorf("load config %q: %w", configFile, err)
	}

	cfg.Prepare()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %q: %w", configFile, err)
	}

	platformresponse.InstallHTTPHandlers()

	// Request ID must be established before logging, recovery, and response shaping.
	// Recovery wraps the inner chain so panic responses still include security,
	// CORS, body-limit handling, and the effective request ID.
	router := httpmiddleware.WrapRouter(
		restrouter.NewRouter(),
		httpmiddleware.RequestID(httpmiddleware.RequestIDConfig{
			HeaderName: cfg.HTTP.RequestID.HeaderName,
			MaxLength:  cfg.HTTP.RequestID.MaxLength,
		}),
		httpmiddleware.AccessLog(),
		httpmiddleware.Recovery(),
		httpmiddleware.SecurityHeaders(httpmiddleware.SecurityHeadersConfig{
			ContentTypeOptions: cfg.HTTP.SecurityHeaders.ContentTypeOptions,
			FrameOptions:       cfg.HTTP.SecurityHeaders.FrameOptions,
			ReferrerPolicy:     cfg.HTTP.SecurityHeaders.ReferrerPolicy,
		}),
		httpmiddleware.CORS(httpmiddleware.CORSConfig{
			Enabled:          cfg.HTTP.CORS.Enabled,
			AllowedOrigins:   cfg.HTTP.CORS.AllowedOrigins,
			AllowedMethods:   cfg.HTTP.CORS.AllowedMethods,
			AllowedHeaders:   cfg.HTTP.CORS.AllowedHeaders,
			ExposedHeaders:   cfg.HTTP.CORS.ExposedHeaders,
			AllowCredentials: cfg.HTTP.CORS.AllowCredentials,
		}),
		httpmiddleware.BodyLimit(cfg.HTTP.MaxBodyBytes),
	)

	server, err := rest.NewServer(cfg.RestConf, rest.WithRouter(router))
	if err != nil {
		return nil, fmt.Errorf("create rest server: %w", err)
	}

	ctx := svc.NewServiceContext(cfg)
	handler.RegisterHandlers(server, ctx)

	return &App{
		Config: cfg,
		server: server,
	}, nil
}

func (a *App) Start() {
	fmt.Printf("Starting server at %s:%d...\n", a.Config.Host, a.Config.Port)
	a.server.Start()
}

func (a *App) Stop() {
	if a == nil || a.server == nil {
		return
	}

	a.server.Stop()
}
