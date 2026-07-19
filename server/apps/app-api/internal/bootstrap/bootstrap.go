package bootstrap

import (
	"context"
	"fmt"
	"sync"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/config"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/handler"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/foundation/cache"
	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/foundation/httpmiddleware"
	"github.com/oublie6/awesome-zero-platform/server/foundation/readiness"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	restrouter "github.com/zeromicro/go-zero/rest/router"
)

var (
	openMySQL = database.Open
	openRedis = cache.Open
)

type App struct {
	Config   config.Config
	server   *rest.Server
	mysql    database.Handle
	redis    cache.Handle
	stopOnce sync.Once
}

func New(configFile string) (*App, error) {
	ctx := context.Background()

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

	mysqlResource, err := openMySQL(ctx, cfg.MySQL)
	if err != nil {
		return nil, err
	}

	redisClient, err := openRedis(ctx, cfg.Redis)
	if err != nil {
		_ = mysqlResource.Close()
		return nil, err
	}

	checker := readiness.New(cfg.Readiness.Timeout,
		namedProbe{name: "mysql", handle: mysqlResource},
		namedProbe{name: "redis", handle: redisClient},
	)

	svcCtx := svc.NewServiceContext(cfg, mysqlResource, redisClient, checker)
	handler.RegisterHandlers(server, svcCtx)

	return &App{
		Config: cfg,
		server: server,
		mysql:  mysqlResource,
		redis:  redisClient,
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

	a.stopOnce.Do(func() {
		a.server.Stop()
		if a.redis != nil {
			_ = a.redis.Close()
		}
		if a.mysql != nil {
			_ = a.mysql.Close()
		}
	})
}

type namedProbe struct {
	name   string
	handle interface {
		Ping(context.Context) error
	}
}

func (n namedProbe) Name() string { return n.name }
func (n namedProbe) Ping(ctx context.Context) error {
	return n.handle.Ping(ctx)
}
