// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/bootstrap"
)

var (
	configFile     = flag.String("f", "etc/main-api.yaml", "the config file")
	healthcheck    = flag.Bool("healthcheck", false, "check the running app-api liveness endpoint")
	healthcheckURL = flag.String("healthcheck-url", "http://127.0.0.1:8888/health/live", "liveness URL used by -healthcheck")
)

func main() {
	flag.Parse()

	if *healthcheck {
		if err := runHealthcheck(*healthcheckURL); err != nil {
			fmt.Fprintf(os.Stderr, "app-api healthcheck failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	app, err := bootstrap.New(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize app-api: %v\n", err)
		os.Exit(1)
	}

	defer app.Stop()
	app.Start()
}
