package kai

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server wraps the MCP server to provide additional behavior
type Server struct {
	mcpServer  *server.MCPServer
	cfg        *serverConfig
	ready      atomic.Bool
	httpServer *http.Server
}

// ServerOption configures the server
type ServerOption func(*serverConfig)

type serverConfig struct {
	version        string
	requestTimeout time.Duration
	tlsCertFile    string
	tlsKeyFile     string
	metricsEnabled bool
}

// Metrics for the MCP server
var (
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kai_requests_total",
			Help: "Total number of MCP tool requests",
		},
		[]string{"tool", "status"},
	)
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kai_request_duration_seconds",
			Help:    "Duration of MCP tool requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tool"},
	)
	activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kai_active_connections",
			Help: "Number of active SSE connections",
		},
	)
)

func init() {
	prometheus.MustRegister(requestsTotal, requestDuration, activeConnections)
}

// WithVersion sets the server version
func WithVersion(version string) ServerOption {
	return func(c *serverConfig) {
		c.version = version
	}
}

// WithRequestTimeout sets the default timeout for Kubernetes operations
func WithRequestTimeout(timeout time.Duration) ServerOption {
	return func(c *serverConfig) {
		c.requestTimeout = timeout
	}
}

// WithTLS enables TLS for the SSE server
func WithTLS(certFile, keyFile string) ServerOption {
	return func(c *serverConfig) {
		c.tlsCertFile = certFile
		c.tlsKeyFile = keyFile
	}
}

// WithMetrics enables Prometheus metrics endpoint
func WithMetrics(enabled bool) ServerOption {
	return func(c *serverConfig) {
		c.metricsEnabled = enabled
	}
}

// NewServer creates a new MCP server for Kubernetes
func NewServer(opts ...ServerOption) *Server {
	cfg := &serverConfig{
		version:        "0.0.1",
		requestTimeout: 30 * time.Second,
		metricsEnabled: true,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Create the MCP server
	mcpServer := server.NewMCPServer(
		"Kubernetes MCP Server",
		cfg.version,
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	s := &Server{
		mcpServer: mcpServer,
		cfg:       cfg,
	}

	return s
}

// AddTool adds a tool to the MCP server
func (s *Server) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	originalHandler := handler
	handler = func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		toolName := request.Params.Name
		slog.Info("tool request received", slog.String("tool", toolName))

		start := time.Now()
		result, err := originalHandler(ctx, request)
		duration := time.Since(start).Seconds()

		status := "success"
		if err != nil || (result != nil && result.IsError) {
			status = "error"
		}

		slog.Info("tool request completed",
			slog.String("tool", toolName),
			slog.String("status", status),
			slog.Float64("duration_seconds", duration),
		)

		if s.cfg.metricsEnabled {
			requestsTotal.WithLabelValues(toolName, status).Inc()
			requestDuration.WithLabelValues(toolName).Observe(duration)
		}

		return result, err
	}
	s.mcpServer.AddTool(tool, handler)
}

// GetRequestTimeout returns the configured request timeout
func (s *Server) GetRequestTimeout() time.Duration {
	return s.cfg.requestTimeout
}

// SetReady marks the server as ready to accept requests
func (s *Server) SetReady(ready bool) {
	s.ready.Store(ready)
}

// Serve starts the server using stdio transport (for Claude Desktop, etc.)
func (s *Server) Serve() error {
	s.SetReady(true)
	return server.ServeStdio(s.mcpServer)
}

// ServeSSE starts the server using SSE transport (for web clients)
func (s *Server) ServeSSE(addr string) error {
	sseServer := server.NewSSEServer(s.mcpServer)

	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/healthz", s.healthzHandler)
	mux.HandleFunc("/readyz", s.readyzHandler)

	// Metrics endpoint
	if s.cfg.metricsEnabled {
		mux.Handle("/metrics", promhttp.Handler())
	}

	// SSE endpoint
	mux.Handle("/sse", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		activeConnections.Inc()
		defer activeConnections.Dec()
		sseServer.ServeHTTP(w, r)
	}))

	// Message endpoint for SSE
	mux.Handle("/message", sseServer)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // SSE requires no write timeout
		IdleTimeout:  60 * time.Second,
	}

	s.SetReady(true)

	slog.Info("SSE server endpoints",
		slog.String("sse", fmt.Sprintf("http://%s/sse", addr)),
		slog.String("health", fmt.Sprintf("http://%s/healthz", addr)),
		slog.String("ready", fmt.Sprintf("http://%s/readyz", addr)),
		slog.String("metrics", fmt.Sprintf("http://%s/metrics", addr)),
	)

	if s.cfg.tlsCertFile != "" && s.cfg.tlsKeyFile != "" {
		slog.Info("TLS enabled",
			slog.String("cert", s.cfg.tlsCertFile),
			slog.String("key", s.cfg.tlsKeyFile),
		)
		return s.httpServer.ListenAndServeTLS(s.cfg.tlsCertFile, s.cfg.tlsKeyFile)
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.SetReady(false)
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// healthzHandler handles liveness probes
func (s *Server) healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"healthy"}`)); err != nil {
		slog.Warn("failed to write healthz response", slog.String("error", err.Error()))
	}
}

// readyzHandler handles readiness probes
func (s *Server) readyzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.ready.Load() {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"ready"}`)); err != nil {
			slog.Warn("failed to write readyz response", slog.String("error", err.Error()))
		}
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		if _, err := w.Write([]byte(`{"status":"not ready"}`)); err != nil {
			slog.Warn("failed to write readyz response", slog.String("error", err.Error()))
		}
	}
}

// TLSConfig returns a TLS configuration for secure connections
func TLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}
