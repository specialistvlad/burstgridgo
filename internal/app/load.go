package app

import (
	"fmt"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
	"github.com/specialistvlad/burstgridgo/internal/handlers"
	"github.com/specialistvlad/burstgridgo/internal/model"
	"github.com/specialistvlad/burstgridgo/internal/registry"

	prnt "github.com/specialistvlad/burstgridgo/modules/print"
)

func (app *App) LoadModules() error {
	logger := ctxlog.FromContext(app.ctx)
	logger.Debug("Loading modules...", "modules_path", app.config.ModulesPath)

	if app.registry == nil {
		logger.Debug("Registry not found, creating new registry...")
		handlers_storage := handlers.New()

		// --- Module registration section ---
		prnt.RegisterHandler(handlers_storage)
		// --- Module registration section ---

		app.registry = registry.New(handlers_storage)
		logger.Debug("Registry created successfully.")
	} else {
		logger.Debug("Using pre-configured registry.")
	}

	return app.registry.LoadGridsRecursively(app.ctx, app.config.ModulesPath)
}

func (app *App) LoadGrids() error {
	logger := ctxlog.FromContext(app.ctx)
	logger.Debug("Loading grids...", "grid_path", app.config.GridPath)

	if app.grid == nil {
		// This is a safeguard, as the model should ideally be initialized earlier.
		logger.Debug("App model is nil, initializing.")
		app.grid = model.NewGrid()
	}

	grid, err := model.LoadGridsRecursively(app.ctx, app.config.GridPath)
	if err != nil {
		return fmt.Errorf("failed to load grid: %w", err)
	}

	app.grid = grid
	logger.Info("Grids loaded successfully.", "steps_found", len(grid.Steps))

	return nil
}
