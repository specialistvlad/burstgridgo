package app

import (
	"fmt"
	"net/http"
)

// healthHandler creates an http.Handler that logs requests to the provided logger.
func (a *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	a.logger.Debug("Health check endpoint hit.", "remote_addr", r.RemoteAddr, "path", r.URL.Path)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// startHealthcheckServer initializes and runs the health check HTTP server.
func (a *App) startHealthcheckServer(port int) {
	a.logger.Debug("Configuring health check server.")
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.healthHandler)

	addr := fmt.Sprintf(":%d", port)

	go func() {
		a.logger.Info("🩺 Health check server starting", "address", fmt.Sprintf("http://localhost%s/health", addr))
		if err := http.ListenAndServe(addr, mux); err != nil {
			a.logger.Error("Health check server failed", "error", err)
		}
	}()
}
