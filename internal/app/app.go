package app

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/localsession"
	"github.com/specialistvlad/burstgridgo/internal/model"
	"github.com/specialistvlad/burstgridgo/internal/registry"
	"github.com/specialistvlad/burstgridgo/internal/session"
)

// App encapsulates the application's dependencies, configuration, and lifecycle.
type App struct {
	ctx        context.Context
	outW       io.Writer
	config     *Config
	grid       *model.Grid
	registry   *registry.Registry
	httpServer *http.Server
}

// NewApp is the constructor for the main application. It returns a fully
// initialized App instance, including its own isolated logger and registry.
func NewApp(ctx context.Context, outW io.Writer, appConfig *Config, reg *registry.Registry) *App {
	logger := newLogger(appConfig.LogLevel, appConfig.LogFormat, outW)
	appCtx := ctxlog.WithLogger(ctx, logger)
	logger.Debug("Logger configured successfully.")

	return &App{
		ctx:      appCtx,
		outW:     outW,
		config:   appConfig,
		registry: reg,
	}
}

// Run executes the main application logic based on the provided configuration.
func (app *App) Run() error {
	logger := ctxlog.FromContext(app.ctx)
	logger.Debug("App.Run method started.")

	defer app.Cleanup()
	go app.healthCheckServer()

	logger.Debug("Checking the presence of model...")
	if app.grid == nil {
		logger.Debug("Model is not loaded, loading default model...")
		app.LoadModules()
	}

	if err := app.LoadGrids(); err != nil {
		return fmt.Errorf("failed to load grids: %w", err)
	}

	// This section is now updated to use our new session-based architecture.
	logger.Debug("Initializing session factory for a local run...")
	var factory session.SessionFactory = &localsession.SessionFactory{}

	logger.Debug("Creating new execution session...")
	s, err := factory.NewSession(app.ctx, app.grid, app.registry.Handlers())
	if err != nil {
		return fmt.Errorf("failed to create execution session: %w", err)
	}
	defer s.Close(app.ctx)

	logger.Debug("Retrieving executor from session...")
	exec, err := s.GetExecutor()
	if err != nil {
		return fmt.Errorf("failed to get executor: %w", err)
	}

	logger.Info("🚀 Starting execution...")
	if err := exec.Execute(app.ctx); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	logger.Info("🏁 Execution finished.")
	return nil
}

func (app *App) Cleanup() error {
	logger := ctxlog.FromContext(app.ctx)
	logger.Debug("Closing application resources...")
	app.closeHealthCheckServer()
	logger.Debug("Application resources closed successfully.")
	return nil
}

// Registry returns the application's registry. This is primarily for integration testing.
func (a *App) Registry() *registry.Registry {
	return a.registry
}

// Grid returns the application's parsed grid model. This is primarily for integration testing.
func (a *App) Grid() *model.Grid {
	return a.grid
}
