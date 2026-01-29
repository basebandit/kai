package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/basebandit/kai/tools"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// CLI flags
	var (
		kubeconfig  string
		contextName string
		transport   string
		sseAddr     string
		showVersion bool
	)

	defaultKubeconfig := filepath.Join(os.Getenv("HOME"), ".kube", "config")

	flag.StringVar(&kubeconfig, "kubeconfig", defaultKubeconfig, "Path to kubeconfig file")
	flag.StringVar(&contextName, "context", "local", "Name for the loaded context")
	flag.StringVar(&transport, "transport", "stdio", "Transport mode: stdio (default) or sse")
	flag.StringVar(&sseAddr, "sse-addr", ":8080", "Address for SSE server (only used with -transport=sse)")
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	if showVersion {
		log.Printf("kai version %s (commit: %s, built: %s)", version, commit, date)
		os.Exit(0)
	}

	// Initialize cluster manager
	cm := cluster.New()

	if err := cm.LoadKubeConfig(contextName, kubeconfig); err != nil {
		log.Fatalf("Failed to load kubeconfig: %v", err)
	}

	log.Printf("Loaded kubeconfig from %s as context %q", kubeconfig, contextName)

	// Create and configure server
	s := kai.NewServer(
		kai.WithVersion(version),
	)

	registerAllTools(s, cm)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)

	go func() {
		switch transport {
		case "sse":
			log.Printf("Starting SSE server on %s", sseAddr)
			errChan <- s.ServeSSE(sseAddr)
		default:
			log.Print("Starting stdio server")
			errChan <- s.Serve()
		}
	}()

	select {
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	}

	log.Print("Server stopped")
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
}
