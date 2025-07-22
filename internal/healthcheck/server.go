package healthcheck

import (
	"fmt"
	"log/slog"
	"net/http"
)

// HealthHandler is a simple handler that responds with HTTP 200 OK.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// StartServer starts a lightweight HTTP server in a separate goroutine.
func StartServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthHandler)

	addr := fmt.Sprintf(":%d", port)

	go func() {
		slog.Info("🩺 Health check server starting", "address", fmt.Sprintf("http://localhost%s/health", addr))
		if err := http.ListenAndServe(addr, mux); err != nil {
			slog.Error("Health check server failed", "error", err)
		}
	}()
}
