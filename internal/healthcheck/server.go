package healthcheck

import (
	"fmt"
	"log"
	"net/http"
)

// HealthHandler is a simple handler that responds with HTTP 200 OK.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

// StartServer starts a lightweight HTTP server in a separate goroutine
// to serve the health check endpoint without blocking the main application.
func StartServer(port int) {
	// Create a new ServeMux to avoid using the default one, which is a best practice
	// to prevent accidentally exposing other handlers.
	mux := http.NewServeMux()
	mux.HandleFunc("/health", HealthHandler)

	addr := fmt.Sprintf(":%d", port)

	// Run the server in a goroutine so it doesn't block.
	go func() {
		log.Printf("🩺 Health check server starting on http://localhost%s/health", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("❗️ Health check server failed: %v", err)
		}
	}()
}
