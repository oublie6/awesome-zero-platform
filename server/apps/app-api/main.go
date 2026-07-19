// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/oublie6/awesome-zero-platform/server/apps/app-api/internal/bootstrap"
)

var configFile = flag.String("f", "etc/main-api.yaml", "the config file")

func main() {
	flag.Parse()

	app, err := bootstrap.New(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize app-api: %v\n", err)
		os.Exit(1)
	}

	defer app.Stop()
	app.Start()
}
