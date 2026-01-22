package cluster

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// ConfigMap represents a Kubernetes ConfigMap resource.
type ConfigMap struct {
	Name        string
	Namespace   string
	Data        map[string]interface{}
	BinaryData  map[string]interface{}
	Labels      map[string]interface{}
	Annotations map[string]interface{}
}

// Create creates a new ConfigMap in the specified namespace.
func (c *ConfigMap) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if err := c.validate(); err != nil {
		return result, err
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, c.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %w", c.Namespace, err)
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
		},
	}

	if c.Data != nil {
		configMap.Data = convertToStringMap(c.Data)
	}

	if c.BinaryData != nil {
		configMap.BinaryData = convertToBinaryDataMap(c.BinaryData)
	}

	if c.Labels != nil {
		labels := convertToStringMap(c.Labels)
		if len(labels) > 0 {
			configMap.ObjectMeta.Labels = labels
		}
	}

	if c.Annotations != nil {
		annotations := convertToStringMap(c.Annotations)
		if len(annotations) > 0 {
			configMap.ObjectMeta.Annotations = annotations
		}
	}

	createdConfigMap, err := client.CoreV1().ConfigMaps(c.Namespace).Create(timeoutCtx, configMap, metav1.CreateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	result = fmt.Sprintf("ConfigMap %q created successfully in namespace %q", createdConfigMap.Name, createdConfigMap.Namespace)
	return result, nil
}

// Get retrieves a ConfigMap by name from the specified namespace.
func (c *ConfigMap) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	var configMap *corev1.ConfigMap
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		return !strings.Contains(err.Error(), "not found")
	}, func() error {
		var getErr error
		configMap, getErr = client.CoreV1().ConfigMaps(c.Namespace).Get(ctx, c.Name, metav1.GetOptions{})
		return getErr
	})

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, fmt.Errorf("ConfigMap %q not found in namespace %q", c.Name, c.Namespace)
		}
		return result, fmt.Errorf("failed to get ConfigMap %q: %v", c.Name, err)
	}

	return formatConfigMap(configMap), nil
}

// List retrieves all ConfigMaps matching the specified criteria.
func (c *ConfigMap) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
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

	var configMaps *corev1.ConfigMapList
	if allNamespaces {
		configMaps, err = client.CoreV1().ConfigMaps("").List(timeoutCtx, listOptions)
	} else {
		configMaps, err = client.CoreV1().ConfigMaps(c.Namespace).List(timeoutCtx, listOptions)
	}

	if err != nil {
		return result, fmt.Errorf("failed to list ConfigMaps: %w", err)
	}

	if len(configMaps.Items) == 0 {
		if labelSelector != "" {
			return result, errors.New("no ConfigMaps found matching the specified label selector")
		}
		if allNamespaces {
			return result, errors.New("no ConfigMaps found in any namespace")
		}
		return result, fmt.Errorf("no ConfigMaps found in namespace %q", c.Namespace)
	}

	return formatConfigMapList(configMaps, allNamespaces), nil
}

// Delete removes a ConfigMap by name from the specified namespace.
func (c *ConfigMap) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if c.Name == "" {
		return result, errors.New("ConfigMap name is required for deletion")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err = client.CoreV1().ConfigMaps(c.Namespace).Get(timeoutCtx, c.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("ConfigMap %q not found in namespace %q: %w", c.Name, c.Namespace, err)
	}

	deleteOptions := metav1.DeleteOptions{}
	err = client.CoreV1().ConfigMaps(c.Namespace).Delete(timeoutCtx, c.Name, deleteOptions)
	if err != nil {
		return result, fmt.Errorf("failed to delete ConfigMap %q: %w", c.Name, err)
	}

	result = fmt.Sprintf("ConfigMap %q deleted successfully from namespace %q", c.Name, c.Namespace)
	return result, nil
}

// Update modifies an existing ConfigMap with the provided data.
func (c *ConfigMap) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if c.Name == "" {
		return result, errors.New("ConfigMap name is required for update")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	existingConfigMap, err := client.CoreV1().ConfigMaps(c.Namespace).Get(timeoutCtx, c.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("ConfigMap %q not found in namespace %q: %w", c.Name, c.Namespace, err)
	}

	if c.Data != nil {
		existingConfigMap.Data = convertToStringMap(c.Data)
	}

	if c.BinaryData != nil {
		existingConfigMap.BinaryData = convertToBinaryDataMap(c.BinaryData)
	}

	if c.Labels != nil {
		labels := convertToStringMap(c.Labels)
		if len(labels) > 0 {
			existingConfigMap.ObjectMeta.Labels = labels
		}
	}

	if c.Annotations != nil {
		annotations := convertToStringMap(c.Annotations)
		if len(annotations) > 0 {
			existingConfigMap.ObjectMeta.Annotations = annotations
		}
	}

	updatedConfigMap, err := client.CoreV1().ConfigMaps(c.Namespace).Update(timeoutCtx, existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to update ConfigMap %q: %w", c.Name, err)
	}

	result = fmt.Sprintf("ConfigMap %q updated successfully in namespace %q", updatedConfigMap.Name, updatedConfigMap.Namespace)
	return result, nil
}

func (c *ConfigMap) validate() error {
	if c.Name == "" {
		return errors.New("ConfigMap name is required")
	}
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}
	return nil
}

func convertToBinaryDataMap(input map[string]interface{}) map[string][]byte {
	if input == nil {
		return nil
	}

	result := make(map[string][]byte, len(input))
	for k, v := range input {
		switch val := v.(type) {
		case string:
			result[k] = []byte(val)
		case []byte:
			result[k] = val
		default:
			result[k] = []byte(fmt.Sprintf("%v", v))
		}
	}
	return result
}
