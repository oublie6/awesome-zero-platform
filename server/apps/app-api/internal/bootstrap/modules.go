package bootstrap

import (
	"context"
	"fmt"
	"strings"
)

// moduleSpec is a deliberately small compile-time composition contract.
// Business-specific dependencies and transport handlers stay inside the
// module adapter; it is not a dynamic plugin API.
type moduleSpec struct {
	name    string
	enabled bool
	start   func() (context.CancelFunc, error)
}

func startModules(modules []moduleSpec) ([]context.CancelFunc, error) {
	seen := make(map[string]struct{}, len(modules))
	stops := make([]context.CancelFunc, 0, len(modules))
	fail := func(err error) ([]context.CancelFunc, error) {
		stopModules(stops)
		return nil, err
	}
	for _, module := range modules {
		if module.name == "" || strings.TrimSpace(module.name) != module.name {
			return fail(fmt.Errorf("invalid business module name %q", module.name))
		}
		if _, exists := seen[module.name]; exists {
			return fail(fmt.Errorf("duplicate business module %q", module.name))
		}
		seen[module.name] = struct{}{}
		if !module.enabled {
			continue
		}
		if module.start == nil {
			return fail(fmt.Errorf("module %q has no start function", module.name))
		}
		stop, err := module.start()
		if err != nil {
			if stop != nil {
				stop()
			}
			return fail(fmt.Errorf("start business module %q: %w", module.name, err))
		}
		if stop == nil {
			return fail(fmt.Errorf("module %q has no shutdown function", module.name))
		}
		stops = append(stops, stop)
	}
	return stops, nil
}

func stopModules(stops []context.CancelFunc) {
	for i := len(stops) - 1; i >= 0; i-- {
		stops[i]()
	}
}
