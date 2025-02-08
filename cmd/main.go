package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/basebandit/kai"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
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
	pods, err := cm.GetPod(context.Background(), "nginx", cm.GetCurrentNamespace())
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(pods)
}

func getPods() {
	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	api := clientset.CoreV1()
	pods, err := api.Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		log.Fatalln("failed to get pods:", err)
	}
	for i, pod := range pods.Items {
		fmt.Printf("[%d] %s\n", i, pod.GetName())
	}
}
