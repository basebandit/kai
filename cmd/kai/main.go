package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/basebandit/kai/tools"
)

const startingServerMsg = "starting server"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// CLI flags
	var (
		kubeconfig     string
		contextName    string
		inCluster      bool
		transport      string
		sseAddr        string
		logFormat      string
		logLevel       string
		tlsCert        string
		tlsKey         string
		requestTimeout time.Duration
		metricsEnabled bool
		showVersion    bool
	)

	defaultKubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	flag.StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig, "Path to kubeconfig file")
	flag.StringVar(&contextName, "context", "local", "Name for the loaded context")
	flag.BoolVar(&inCluster, "in-cluster", false, "Use in-cluster Kubernetes configuration (for running inside a pod)")
	flag.StringVar(&transport, "transport", "stdio", "Transport mode: stdio (default), streamable-http, or sse-legacy. \"sse\" is accepted as a deprecated alias of \"sse-legacy\".")
	flag.StringVar(&sseAddr, "sse-addr", ":8080", "Address for the HTTP listener (used with streamable-http or sse-legacy). The flag name is kept for backwards compatibility.")
	flag.StringVar(&logFormat, "log-format", "json", "Log format: json (default) or text")
	flag.StringVar(&logLevel, "log-level", "info", "Log level: debug, info, warn, error")
	flag.StringVar(&tlsCert, "tls-cert", "", "Path to TLS certificate file (enables HTTPS for SSE)")
	flag.StringVar(&tlsKey, "tls-key", "", "Path to TLS private key file (enables HTTPS for SSE)")
	flag.DurationVar(&requestTimeout, "request-timeout", 30*time.Second, "Timeout for Kubernetes API requests")
	flag.BoolVar(&metricsEnabled, "metrics", true, "Enable Prometheus metrics endpoint at /metrics")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	// Initialize structured logger
	logger := initLogger(logFormat, logLevel)
	slog.SetDefault(logger)

	if showVersion {
		fmt.Printf("kai version %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// Initialize cluster manager
	cm := cluster.New(cluster.WithRequestTimeout(requestTimeout))

	if inCluster {
		if err := cm.LoadInClusterConfig(contextName); err != nil {
			logger.Error(
				"failed to load in-cluster config",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
		logger.Info(
			"in-cluster config loaded",
			slog.String("context", contextName),
		)
	} else {
		if err := cm.LoadKubeConfig(contextName, kubeconfig); err != nil {
			logger.Error(
				"failed to load kubeconfig",
				slog.String("path", kubeconfig),
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
		logger.Info(
			"kubeconfig loaded",
			slog.String("path", kubeconfig),
			slog.String("context", contextName),
		)
	}

	// Create and configure server
	serverOpts := []kai.ServerOption{
		kai.WithVersion(version),
		kai.WithRequestTimeout(requestTimeout),
		kai.WithMetrics(metricsEnabled),
	}

	if tlsCert != "" && tlsKey != "" {
		serverOpts = append(serverOpts, kai.WithTLS(tlsCert, tlsKey))
		logger.Info(
			"TLS enabled",
			slog.String("cert", tlsCert),
			slog.String("key", tlsKey),
		)
	}

	s := kai.NewServer(serverOpts...)

	registerAllTools(s, cm)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)

	go func() {
		switch transport {
		case "streamable-http", "http":
			logger.Info(
				startingServerMsg,
				slog.String("transport", "streamable-http"),
				slog.String("address", sseAddr),
			)
			errChan <- s.ServeStreamableHTTP(sseAddr)
		case "sse":
			logger.Warn("transport \"sse\" is deprecated; use \"sse-legacy\" or migrate to \"streamable-http\"")
			fallthrough
		case "sse-legacy":
			logger.Info(
				startingServerMsg,
				slog.String("transport", "sse-legacy"),
				slog.String("address", sseAddr),
			)
			errChan <- s.ServeSSE(sseAddr)
		case "stdio", "":
			logger.Info(
				startingServerMsg,
				slog.String("transport", "stdio"),
			)
			errChan <- s.Serve()
		default:
			errChan <- fmt.Errorf("unknown transport %q (valid: stdio, streamable-http, sse-legacy)", transport)
		}
	}()

	select {
	case err := <-errChan:
		if err != nil {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case sig := <-sigChan:
		logger.Info("shutdown initiated", slog.String("signal", sig.String()))

		// Graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown error", slog.String("error", err.Error()))
		}
	}

	logger.Info("server stopped")
}

func initLogger(format, level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
	}

	var handler slog.Handler
	switch format {
	case "text":
		handler = slog.NewTextHandler(os.Stderr, opts)
	default:
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	return slog.New(handler)
}

func registerAllTools(s *kai.Server, cm *cluster.Manager) {
	tools.RegisterNamespaceTools(s, cm)
	tools.RegisterPodTools(s, cm)
	tools.RegisterDeploymentTools(s, cm)
	tools.RegisterServiceTools(s, cm)
	tools.RegisterContextTools(s, cm)
	tools.RegisterConfigMapTools(s, cm)
	tools.RegisterSecretTools(s, cm)
	tools.RegisterJobTools(s, cm)
	tools.RegisterCronJobTools(s, cm)
	tools.RegisterIngressTools(s, cm)
	tools.RegisterOperationsTools(s, cm)
	tools.RegisterEventTools(s, cm)
	tools.RegisterNodeTools(s, cm)
	tools.RegisterHealthTools(s, cm)
	tools.RegisterStorageTools(s, cm)
	tools.RegisterRBACTools(s, cm)
	tools.RegisterCustomResourceTools(s, cm)
	tools.RegisterApplyTools(s, cm)
	tools.RegisterDeleteTools(s, cm)
}
