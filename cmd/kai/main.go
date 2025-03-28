package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/basebandit/kai"
	"github.com/basebandit/kai/tools"
)

func main() {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)
	cm := kai.NewClusterManager()

	err := cm.LoadKubeConfig("local", kubeconfig)
	if err != nil {
		log.Fatalln(err)
	}

	s := kai.NewServer()

	registerAllTools(s, cm)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		fmt.Fprintln(os.Stdout, "Server started")
		errChan <- s.Serve()
	}()

	select {
	case err := <-errChan:
		if err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	case sig := <-sigChan:
		fmt.Fprintf(os.Stderr, "Received signal %v, shutting down...\n", sig)
	}
	fmt.Fprintln(os.Stdout, "Server stopped")
}

func registerAllTools(s *kai.Server, cm *kai.ClusterManager) {
	tools.RegisterPodTools(s, cm)
}
