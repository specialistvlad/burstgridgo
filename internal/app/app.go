package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/specialistvlad/burstgridgo/internal/config"
	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/registry"
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
	outW      io.Writer
	logger    *slog.Logger
	registry  *registry.Registry
	config    *config.Model
	converter config.Converter
}

// NewApp is the constructor for the main application. It returns a fully
// initialized App instance, including its own isolated logger and registry.
func NewApp(outW io.Writer, appConfig *AppConfig, loader config.Loader, modules ...registry.Module) *App {
	logger := newLogger(appConfig.LogLevel, appConfig.LogFormat, outW)
	ctx := ctxlog.WithLogger(context.Background(), logger)
	logger.Debug("Logger configured successfully.")

	// Merge all configuration paths into a single collection for the loader.
	var configPaths []string
	if appConfig.GridPath != "" {
		configPaths = append(configPaths, appConfig.GridPath)
	}
	if appConfig.ModulesPath != "" {
		configPaths = append(configPaths, appConfig.ModulesPath)
	}

	// Load all configuration into the format-agnostic model first.
	cfgModel, converter, err := loader.Load(ctx, configPaths...)
	if err != nil {
		// A failure to load config is a fatal startup error.
		panic(fmt.Errorf("failed to load configuration: %w", err))
	}
	logger.Debug("Configuration loaded and translated into unified model.")

	// Create and populate the registry with Go handlers.
	reg := registry.New()
	if len(modules) == 0 {
		modules = coreModules
	}
	for _, mod := range modules {
		mod.Register(reg)
	}
	logger.Debug("All Go modules registered.", "count", len(modules))

	// Populate the registry's definitions from the loaded config model.
	reg.PopulateDefinitionsFromModel(cfgModel)
	logger.Debug("Registry definitions populated from config model.")

	// Validate the integrity of the registry.
	if err := reg.ValidateRegistry(ctx); err != nil {
		// This is a programmer error (mismatch between code and config), so we panic.
		panic(err)
	}
	logger.Debug("Registry validation passed.")

	return &App{
		outW:      outW,
		logger:    logger,
		registry:  reg,
		config:    cfgModel,
		converter: converter,
	}
}

// Registry returns the application's registry. This is primarily for testing.
func (a *App) Registry() *registry.Registry {
	return a.registry
}
