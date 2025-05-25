package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/cluster"
	"github.com/basebandit/kai/tools"
)

func main() {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	cm := cluster.New()

	err := cm.LoadKubeConfig("local", kubeconfig)
	if err != nil {
		log(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	s := kai.NewServer()

	registerAllTools(s, cm)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		log(os.Stdout, "Server started\n")
		errChan <- s.Serve()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			log(os.Stderr, "Server error: %v\n", err)
		}
	case sig := <-sigChan:
		log(os.Stderr, "Received signal %v, shutting down...\n", sig)
	}
	log(os.Stdout, "Server stopped\n")
}

func log(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}

func registerAllTools(s *kai.Server, cm *cluster.Manager) {
	tools.RegisterPodTools(s, cm)
	tools.RegisterDeploymentTools(s, cm)
	tools.RegisterServiceTools(s, cm)
	tools.RegisterContextTools(s, cm)
}
