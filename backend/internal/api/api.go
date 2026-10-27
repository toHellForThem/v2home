// Package api wires the REST endpoints, the WebSocket stream and the SPA.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"v2home/internal/buildinfo"
	"v2home/internal/status"
	"v2home/internal/wsproto"
	"v2home/webui"
)

// Options configures the HTTP API.
type Options struct {
	Logger     *slog.Logger
	Status     *status.Collector
	WSInterval time.Duration
}

type API struct {
	logger     *slog.Logger
	status     *status.Collector
	wsInterval time.Duration
}

// New builds the root HTTP handler: REST API, WebSocket stream and the
// embedded Vue SPA.
func New(opts Options) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	collector := opts.Status
	if collector == nil {
		collector = status.New()
	}
	interval := opts.WSInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}

	a := &API{logger: logger, status: collector, wsInterval: interval}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", a.handleHealth)
	mux.HandleFunc("GET /api/status", a.handleStatus)
	mux.HandleFunc("GET /api/version", a.handleVersion)
	mux.HandleFunc("GET /ws/status", a.handleStatusWS)
	mux.HandleFunc("/api/", a.handleAPINotFound)
	mux.Handle("/", webui.Handler())

	return a.withMiddleware(mux)
}

func (a *API) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		defer func() {
			if recovered := recover(); recovered != nil {
				a.logger.Error("panic recovered", "panic", recovered, "method", r.Method, "path", r.URL.Path)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
			a.logger.Debug("request",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", time.Since(start),
			)
		}()

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"uptimeSeconds": int64(time.Since(a.status.StartedAt()).Seconds()),
	})
}

func (a *API) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.status.Snapshot())
}

func (a *API) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"app":       "v2home",
		"version":   buildinfo.Version,
		"commit":    buildinfo.Commit,
		"buildTime": buildinfo.BuildTime,
		"goVersion": buildinfo.GoVersion(),
	})
}

func (a *API) handleAPINotFound(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, map[string]string{
		"error": "not found",
		"path":  r.URL.Path,
	})
}

func (a *API) handleStatusWS(w http.ResponseWriter, r *http.Request) {
	conn, err := wsproto.Upgrade(w, r)
	if err != nil {
		a.logger.Warn("websocket upgrade failed", "err", err, "remote", r.RemoteAddr)
		return
	}
	defer conn.Close()

	a.logger.Debug("websocket client connected", "remote", r.RemoteAddr)

	sendSnapshot := func() error {
		payload, err := json.Marshal(a.status.Snapshot())
		if err != nil {
			return err
		}
		return conn.WriteText(payload)
	}

	if err := sendSnapshot(); err != nil {
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			opcode, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}
			switch opcode {
			case wsproto.OpClose:
				return
			case wsproto.OpPing:
				if err := conn.WritePong(payload); err != nil {
					return
				}
			}
		}
	}()

	ticker := time.NewTicker(a.wsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := sendSnapshot(); err != nil {
				return
			}
		}
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
