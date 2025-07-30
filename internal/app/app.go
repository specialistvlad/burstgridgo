package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/vk/burstgridgo/internal/ctxlog"
	"github.com/vk/burstgridgo/internal/registry"
)

// AppConfig holds all the necessary configuration for an App instance to run.
type AppConfig struct {
	GridPath        string
	ModulesPath     string
	HealthcheckPort int
	LogFormat       string
	LogLevel        string
	WorkerCount     int
}

// App encapsulates the application's dependencies, configuration, and lifecycle.
type App struct {
	outW     io.Writer
	logger   *slog.Logger
	registry *registry.Registry
}

// NewApp is the constructor for the main application. It returns a fully
// initialized App instance, including its own isolated logger and registry.
// It accepts an optional list of modules to register, which is used for testing.
func NewApp(outW io.Writer, appConfig *AppConfig, modules ...registry.Module) *App {
	logger := newLogger(appConfig.LogLevel, appConfig.LogFormat, outW)
	logger.Debug("Logger configured successfully.")

	reg := registry.New()
	// If no modules are passed, use the default core modules.
	if len(modules) == 0 {
		modules = coreModules
	}

	for _, mod := range modules {
		mod.Register(reg)
	}
	logger.Debug("All Go modules registered.", "count", len(modules))

	// Discover HCL module definitions from the filesystem *before* validation.
	ctx := ctxlog.WithLogger(context.Background(), logger)
	if appConfig.ModulesPath != "" {
		logger.Debug("Starting module discovery...", "path", appConfig.ModulesPath)
		if err := DiscoverModules(ctx, appConfig.ModulesPath, reg); err != nil {
			// A failure to discover modules is a fatal startup error.
			panic(fmt.Errorf("failed to discover modules: %w", err))
		}
		logger.Debug("HCL module discovery complete.")
	}

	// Validate the integrity of the registry after all definitions are loaded.
	if err := reg.ValidateRegistry(); err != nil {
		// This is a programmer error (a mismatch between code and HCL), so we panic.
		panic(err)
	}
	logger.Debug("Registry validation passed.")

	return &App{
		outW:     outW,
		logger:   logger,
		registry: reg,
	}
}

// Registry returns the application's registry. This is primarily for testing.
func (a *App) Registry() *registry.Registry {
	return a.registry
}
