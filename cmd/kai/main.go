package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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

	fmt.Fprintln(os.Stdout, "Server started")
	if err := s.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
	}
	fmt.Fprintln(os.Stdout, "Server stopped")
}

func registerAllTools(s *kai.Server, cm *kai.ClusterManager) {
	tools.RegisterPodTools(s, cm)
}
