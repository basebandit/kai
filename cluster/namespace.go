package cluster

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

type Namespace struct {
	Name        string
	Labels      map[string]interface{}
	Annotations map[string]interface{}
}

const (
	defaultTimeout = 30 * time.Second
	listTimeout    = 20 * time.Second
)

func (n *Namespace) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if err := n.validate(); err != nil {
		return result, err
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: n.Name,
		},
	}

	if n.Labels != nil {
		labels := convertToStringMap(n.Labels)
		if len(labels) > 0 {
			namespace.ObjectMeta.Labels = labels
		}
	}

	if n.Annotations != nil {
		annotations := convertToStringMap(n.Annotations)
		if len(annotations) > 0 {
			namespace.ObjectMeta.Annotations = annotations
		}
	}

	createdNamespace, err := client.CoreV1().Namespaces().Create(timeoutCtx, namespace, metav1.CreateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to create namespace: %w", err)
	}

	result = fmt.Sprintf("Namespace %q created successfully", createdNamespace.Name)
	return result, nil
}

func (n *Namespace) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string
	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, err
	}

	var namespace *corev1.Namespace
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		return !strings.Contains(err.Error(), "not found")
	}, func() error {
		var getErr error
		namespace, getErr = client.CoreV1().Namespaces().Get(ctx, n.Name, metav1.GetOptions{})
		return getErr
	})

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, fmt.Errorf("namespace '%s' not found", n.Name)
		}
		return result, fmt.Errorf("failed to get namespace '%s': %v", n.Name, err)
	}

	return formatNamespace(namespace), nil
}

func (n *Namespace) List(ctx context.Context, cm kai.ClusterManager, labelSelector string) (string, error) {
	var result string
	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	namespaces, err := client.CoreV1().Namespaces().List(timeoutCtx, listOptions)
	if err != nil {
		return result, fmt.Errorf("failed to list namespaces: %w", err)
	}

	if len(namespaces.Items) == 0 {
		if labelSelector != "" {
			return result, errors.New("no namespaces found matching the specified selectors")
		}
		return result, errors.New("no namespaces found")
	}

	return formatNamespaceList(namespaces, labelSelector), nil
}

func (n *Namespace) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if n.Name != "" {
		_, err = client.CoreV1().Namespaces().Get(timeoutCtx, n.Name, metav1.GetOptions{})
		if err != nil {
			return result, fmt.Errorf("failed to find namespace %q: %w", n.Name, err)
		}

		deleteOptions := metav1.DeleteOptions{}
		err = client.CoreV1().Namespaces().Delete(timeoutCtx, n.Name, deleteOptions)
		if err != nil {
			return result, fmt.Errorf("failed to delete namespace %q: %w", n.Name, err)
		}

		result = fmt.Sprintf("Namespace %q deleted successfully", n.Name)
		return result, nil
	}

	if len(n.Labels) > 0 {
		labelSelector := ""
		for k, v := range n.Labels {
			if labelSelector != "" {
				labelSelector += ","
			}
			switch val := v.(type) {
			case string:
				labelSelector += fmt.Sprintf("%s=%s", k, val)
			default:
				labelSelector += fmt.Sprintf("%s=%v", k, val)
			}
		}

		listOptions := metav1.ListOptions{
			LabelSelector: labelSelector,
		}

		namespaceList, err := client.CoreV1().Namespaces().List(timeoutCtx, listOptions)
		if err != nil {
			return result, fmt.Errorf("failed to list namespaces with label selector %q: %w", labelSelector, err)
		}

		if len(namespaceList.Items) == 0 {
			return result, fmt.Errorf("no namespaces found with label selector %q", labelSelector)
		}

		deleteOptions := metav1.DeleteOptions{}
		deletedCount := 0
		deletedNames := []string{}

		for _, namespace := range namespaceList.Items {
			err = client.CoreV1().Namespaces().Delete(timeoutCtx, namespace.Name, deleteOptions)
			if err != nil {
				result += fmt.Sprintf("Failed to delete namespace %q: %v\n", namespace.Name, err)
			} else {
				deletedCount++
				deletedNames = append(deletedNames, namespace.Name)
			}
		}

		if deletedCount == 0 {
			return result, fmt.Errorf("failed to delete any namespaces with label selector %q", labelSelector)
		}

		result = fmt.Sprintf("Deleted %d namespaces with label selector %q:\n- %s",
			deletedCount, labelSelector, strings.Join(deletedNames, "\n- "))
		return result, nil
	}

	return result, errors.New("either namespace name or label selector must be provided")
}

// Update updates an existing namespace
func (n *Namespace) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	namespace, err := client.CoreV1().Namespaces().Get(timeoutCtx, n.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get namespace: %w", err)
	}

	if len(n.Labels) > 0 {
		if namespace.Labels == nil {
			namespace.Labels = make(map[string]string)
		}
		for k, v := range convertToStringMap(n.Labels) {
			namespace.Labels[k] = v
		}
	}

	if len(n.Annotations) > 0 {
		if namespace.Annotations == nil {
			namespace.Annotations = make(map[string]string)
		}
		for k, v := range convertToStringMap(n.Annotations) {
			namespace.Annotations[k] = v
		}
	}

	updatedNamespace, err := client.CoreV1().Namespaces().Update(timeoutCtx, namespace, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to update namespace: %w", err)
	}

	result = fmt.Sprintf("Namespace %q updated successfully", updatedNamespace.Name)
	return result, nil
}

func (n *Namespace) validate() error {
	if n.Name == "" {
		return errors.New("namespace name is required")
	}
	return nil
}
