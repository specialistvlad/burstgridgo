package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/specialistvlad/burstgridgo/internal/ctxlog"
)

// healthHandler creates an http.Handler that logs requests to the provided logger.
func (app *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	logger := ctxlog.FromContext(app.ctx)
	logger.Debug("Health check endpoint hit.", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// healthCheckServer initializes and runs the health check HTTP server.
func (app *App) healthCheckServer() {
	logger := ctxlog.FromContext(app.ctx)
	logger.Debug("Configuring health check server.")
	if app.config.HealthcheckPort <= 0 {
		logger.Warn("Health check server not started: disabled")
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", app.healthHandler)

	addr := fmt.Sprintf(":%d", app.config.HealthcheckPort)

	// Create the server instance and store it on the app struct.
	app.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Run the server in a goroutine so it doesn't block.
	go func() {
		logger.Info("🩺 Health check server starting", "address", fmt.Sprintf("http://localhost%s/health", addr))
		// ListenAndServe will return an error on graceful shutdown.
		// We check for this specific error to avoid logging a false positive.
		if err := app.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Health check server failed unexpectedly", "error", err)
		}
	}()
}

func (app *App) closeHealthCheckServer() error {
	logger := ctxlog.FromContext(app.ctx)
	logger.Debug("Closing health check server...")

	if app.httpServer == nil {
		logger.Debug("Health check server was not running.")
		return nil
	}

	// Create a context with a timeout for the shutdown process.
	ctx, cancel := context.WithTimeout(app.ctx, 5*time.Second)
	defer cancel()

	logger.Info("🩺 Shutting down health check server...")
	if err := app.httpServer.Shutdown(ctx); err != nil {
		logger.Error("Health check server shutdown failed", "error", err)
		return err
	}

	logger.Debug("Health check server shut down gracefully.")
	return nil
}
