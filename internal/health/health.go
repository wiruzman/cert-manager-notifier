package health

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	healthy int64 = 1
)

// HealthServer provides health check endpoints
type HealthServer struct {
	port int
}

// NewHealthServer creates a new health server
func NewHealthServer(port int) *HealthServer {
	return &HealthServer{port: port}
}

// Start starts the health check server
func (h *HealthServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.healthHandler)
	mux.HandleFunc("/ready", h.readyHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", h.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return server.ListenAndServe()
}

// healthHandler handles liveness probe requests
func (h *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt64(&healthy) == 1 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("Service Unavailable"))
	}
}

// readyHandler handles readiness probe requests
func (h *HealthServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	if atomic.LoadInt64(&healthy) == 1 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Ready"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("Not Ready"))
	}
}

// SetHealthy sets the health status
func SetHealthy(status bool) {
	if status {
		atomic.StoreInt64(&healthy, 1)
	} else {
		atomic.StoreInt64(&healthy, 0)
	}
}
