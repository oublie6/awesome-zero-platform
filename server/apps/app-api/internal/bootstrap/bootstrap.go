package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/adminhttp"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/config"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/handler"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/securityhttp"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/oublie6/awesome-zero-platform/server/foundation/cache"
	"github.com/oublie6/awesome-zero-platform/server/foundation/database"
	"github.com/oublie6/awesome-zero-platform/server/foundation/httpmiddleware"
	"github.com/oublie6/awesome-zero-platform/server/foundation/observability"
	"github.com/oublie6/awesome-zero-platform/server/foundation/readiness"
	platformresponse "github.com/oublie6/awesome-zero-platform/server/foundation/response"
	"github.com/oublie6/awesome-zero-platform/server/platform/admin"
	"github.com/oublie6/awesome-zero-platform/server/platform/admin/mysqlstore"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn/adapter/identityprovider"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn/adapter/jwthmac"
	"github.com/oublie6/awesome-zero-platform/server/platform/authn/adapter/redissession"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz"
	"github.com/oublie6/awesome-zero-platform/server/platform/authz/adapter/casbinmysql"
	"github.com/oublie6/awesome-zero-platform/server/platform/identity"
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
	authz    *casbinmysql.Engine
	stopOnce sync.Once
}

func New(configFile string) (*App, error) {
	ctx := context.Background()

	var cfg config.Config
	if err := conf.Load(configFile, &cfg); err != nil {
		return nil, fmt.Errorf("load config %q: %w", configFile, err)
	}

	cfg.Prepare()
	if err := cfg.ApplyEnvironment(); err != nil {
		return nil, fmt.Errorf("apply environment configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %q: %w", configFile, err)
	}

	platformresponse.InstallHTTPHandlers()

	var metrics *observability.Metrics
	metricsMiddleware := func(next http.Handler) http.Handler { return next }
	if cfg.Observability.Metrics.Enabled {
		metrics = observability.NewMetrics(cfg.Observability.Metrics.Namespace)
		metricsMiddleware = metrics.Middleware
	}

	router := httpmiddleware.WrapRouter(
		restrouter.NewRouter(),
		httpmiddleware.RequestID(httpmiddleware.RequestIDConfig{HeaderName: cfg.HTTP.RequestID.HeaderName, MaxLength: cfg.HTTP.RequestID.MaxLength}),
		httpmiddleware.AccessLog(),
		httpmiddleware.Recovery(),
		metricsMiddleware,
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

	var authorizationEngine *casbinmysql.Engine
	closeResources := func() {
		if authorizationEngine != nil {
			_ = authorizationEngine.Close()
		}
		_ = redisClient.Close()
		_ = mysqlResource.Close()
	}

	checker := readiness.New(cfg.Readiness.Timeout,
		namedProbe{name: "mysql", handle: mysqlResource},
		namedProbe{name: "redis", handle: redisClient},
	)

	svcCtx := svc.NewServiceContext(cfg, mysqlResource, redisClient, checker)
	svcCtx.Metrics = metrics

	var identityModelCache *cache.ModelCache
	var adminModelCache *cache.ModelCache
	if mysqlResource.DB() != nil {
		identityModelCache, err = cache.NewModelCache(cfg.Redis, "identity:account", identity.ErrAccountNotFound)
		if err != nil {
			closeResources()
			return nil, fmt.Errorf("initialize identity model cache: %w", err)
		}
		adminModelCache, err = cache.NewModelCache(cfg.Redis, "admin", admin.ErrNotFound)
		if err != nil {
			closeResources()
			return nil, fmt.Errorf("initialize admin model cache: %w", err)
		}
		svcCtx.Identity = identity.NewServiceWithCache(mysqlResource, identityModelCache)
	}

	var sessionAdmin authn.SessionAdministrator
	if cfg.Authentication.Enabled {
		codec, err := jwthmac.New(cfg.Authentication.AccessTokenSecret, cfg.Authentication.Issuer)
		if err != nil {
			closeResources()
			return nil, fmt.Errorf("initialize access token codec: %w", err)
		}
		sessionStore, err := redissession.New(redisClient.Client(), cfg.Authentication.SessionKeyPrefix)
		if err != nil {
			closeResources()
			return nil, fmt.Errorf("initialize session store: %w", err)
		}
		authentication, err := authn.NewService(
			identityprovider.New(svcCtx.Identity),
			codec,
			sessionStore,
			authn.Config{AccessTTL: cfg.Authentication.AccessTTL, RefreshTTL: cfg.Authentication.RefreshTTL},
		)
		if err != nil {
			closeResources()
			return nil, fmt.Errorf("initialize authentication service: %w", err)
		}
		svcCtx.Authn = authentication
		svcCtx.SessionAdmin = sessionStore
		sessionAdmin = sessionStore
	}

	var authorizationAdmin authz.Administrator
	if cfg.Authorization.Enabled {
		authorizationEngine, err = casbinmysql.NewClustered(
			mysqlResource.DB(),
			redisClient.Client(),
			casbinmysql.ClusterConfig{
				Enabled:        cfg.Authorization.Cluster.Enabled,
				InstanceID:     cfg.Authorization.Cluster.InstanceID,
				Channel:        cfg.Authorization.Cluster.Channel,
				PollInterval:   cfg.Authorization.Cluster.PollInterval,
				PublishTimeout: cfg.Authorization.Cluster.PublishTimeout,
				ReloadTimeout:  cfg.Authorization.Cluster.ReloadTimeout,
			},
		)
		if err != nil {
			closeResources()
			return nil, fmt.Errorf("initialize casbin authorization: %w", err)
		}
		if err := authorizationEngine.Start(ctx); err != nil {
			closeResources()
			return nil, fmt.Errorf("start casbin authorization synchronization: %w", err)
		}
		checker.AddProbe(authorizationEngine)

		authorization, err := authz.NewService(authorizationEngine, authorizationEngine)
		if err != nil {
			closeResources()
			return nil, fmt.Errorf("initialize authorization service: %w", err)
		}
		svcCtx.Authz = authorization
		svcCtx.Authorizer = authorization
		svcCtx.AuthzAdmin = authorizationEngine
		authorizationAdmin = authorizationEngine
	}

	if cfg.Admin.Enabled {
		store, err := mysqlstore.NewCached(mysqlResource.DB(), adminModelCache)
		if err != nil {
			closeResources()
			return nil, fmt.Errorf("initialize admin store: %w", err)
		}
		adminService, err := admin.NewService(
			svcCtx.Identity,
			authorizationAdmin,
			sessionAdmin,
			store,
			admin.Config{BootstrapToken: cfg.Admin.BootstrapToken},
		)
		if err != nil {
			closeResources()
			return nil, fmt.Errorf("initialize admin service: %w", err)
		}
		svcCtx.Admin = adminService
	}

	handler.RegisterHandlers(server, svcCtx)
	securityhttp.Register(server, svcCtx)
	adminhttp.Register(server, svcCtx)
	if metrics != nil {
		server.AddRoutes([]rest.Route{{Method: http.MethodGet, Path: cfg.Observability.Metrics.Path, Handler: metrics.Handler().ServeHTTP}})
	}

	return &App{
		Config: cfg,
		server: server,
		mysql:  mysqlResource,
		redis:  redisClient,
		authz:  authorizationEngine,
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
		if a.authz != nil {
			_ = a.authz.Close()
		}
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
	handle interface{ Ping(context.Context) error }
}

func (n namedProbe) Name() string { return n.name }
func (n namedProbe) Ping(ctx context.Context) error {
	return n.handle.Ping(ctx)
}
