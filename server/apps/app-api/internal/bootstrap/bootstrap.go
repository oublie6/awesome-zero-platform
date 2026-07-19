package bootstrap

import (
	"fmt"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/config"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/handler"
	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/svc"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
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

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config %q: %w", configFile, err)
	}

	server, err := rest.NewServer(cfg.RestConf)
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
