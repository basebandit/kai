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

// Secret represents a Kubernetes Secret resource.
type Secret struct {
	Name        string
	Namespace   string
	Type        string
	Data        map[string]interface{}
	StringData  map[string]interface{}
	Labels      map[string]interface{}
	Annotations map[string]interface{}
}

// Create creates a new Secret in the specified namespace.
func (s *Secret) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if err := s.validate(); err != nil {
		return result, err
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, s.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %w", s.Namespace, err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
		Type: corev1.SecretType(s.Type),
	}

	if s.Type == "" {
		secret.Type = corev1.SecretTypeOpaque
	}

	if s.Data != nil {
		secret.Data = convertToSecretDataMap(s.Data)
	}

	if s.StringData != nil {
		secret.StringData = convertToStringMap(s.StringData)
	}

	if s.Labels != nil {
		labels := convertToStringMap(s.Labels)
		if len(labels) > 0 {
			secret.ObjectMeta.Labels = labels
		}
	}

	if s.Annotations != nil {
		annotations := convertToStringMap(s.Annotations)
		if len(annotations) > 0 {
			secret.ObjectMeta.Annotations = annotations
		}
	}

	createdSecret, err := client.CoreV1().Secrets(s.Namespace).Create(timeoutCtx, secret, metav1.CreateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to create Secret: %w", err)
	}

	result = fmt.Sprintf("Secret %q created successfully in namespace %q", createdSecret.Name, createdSecret.Namespace)
	return result, nil
}

// Get retrieves a Secret by name from the specified namespace.
func (s *Secret) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	var secret *corev1.Secret
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		return !strings.Contains(err.Error(), "not found")
	}, func() error {
		var getErr error
		secret, getErr = client.CoreV1().Secrets(s.Namespace).Get(ctx, s.Name, metav1.GetOptions{})
		return getErr
	})

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, fmt.Errorf("Secret %q not found in namespace %q", s.Name, s.Namespace)
		}
		return result, fmt.Errorf("failed to get Secret %q: %v", s.Name, err)
	}

	return formatSecret(secret), nil
}

// List retrieves all Secrets matching the specified criteria.
func (s *Secret) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
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

	var secrets *corev1.SecretList
	if allNamespaces {
		secrets, err = client.CoreV1().Secrets("").List(timeoutCtx, listOptions)
	} else {
		secrets, err = client.CoreV1().Secrets(s.Namespace).List(timeoutCtx, listOptions)
	}

	if err != nil {
		return result, fmt.Errorf("failed to list Secrets: %w", err)
	}

	if len(secrets.Items) == 0 {
		if labelSelector != "" {
			return result, errors.New("no Secrets found matching the specified label selector")
		}
		if allNamespaces {
			return result, errors.New("no Secrets found in any namespace")
		}
		return result, fmt.Errorf("no Secrets found in namespace %q", s.Namespace)
	}

	return formatSecretList(secrets, allNamespaces), nil
}

// Delete removes a Secret by name from the specified namespace.
func (s *Secret) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if s.Name == "" {
		return result, errors.New("Secret name is required for deletion")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err = client.CoreV1().Secrets(s.Namespace).Get(timeoutCtx, s.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("Secret %q not found in namespace %q: %w", s.Name, s.Namespace, err)
	}

	deleteOptions := metav1.DeleteOptions{}
	err = client.CoreV1().Secrets(s.Namespace).Delete(timeoutCtx, s.Name, deleteOptions)
	if err != nil {
		return result, fmt.Errorf("failed to delete Secret %q: %w", s.Name, err)
	}

	result = fmt.Sprintf("Secret %q deleted successfully from namespace %q", s.Name, s.Namespace)
	return result, nil
}

// Update modifies an existing Secret with the provided data.
func (s *Secret) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if s.Name == "" {
		return result, errors.New("Secret name is required for update")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	existingSecret, err := client.CoreV1().Secrets(s.Namespace).Get(timeoutCtx, s.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("Secret %q not found in namespace %q: %w", s.Name, s.Namespace, err)
	}

	if s.Data != nil {
		existingSecret.Data = convertToSecretDataMap(s.Data)
	}

	if s.StringData != nil {
		existingSecret.StringData = convertToStringMap(s.StringData)
	}

	if s.Type != "" {
		existingSecret.Type = corev1.SecretType(s.Type)
	}

	if s.Labels != nil {
		labels := convertToStringMap(s.Labels)
		if len(labels) > 0 {
			existingSecret.ObjectMeta.Labels = labels
		}
	}

	if s.Annotations != nil {
		annotations := convertToStringMap(s.Annotations)
		if len(annotations) > 0 {
			existingSecret.ObjectMeta.Annotations = annotations
		}
	}

	updatedSecret, err := client.CoreV1().Secrets(s.Namespace).Update(timeoutCtx, existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to update Secret %q: %w", s.Name, err)
	}

	result = fmt.Sprintf("Secret %q updated successfully in namespace %q", updatedSecret.Name, updatedSecret.Namespace)
	return result, nil
}

func (s *Secret) validate() error {
	if s.Name == "" {
		return errors.New("Secret name is required")
	}
	if s.Namespace == "" {
		return errors.New("namespace is required")
	}
	return nil
}

func convertToSecretDataMap(input map[string]interface{}) map[string][]byte {
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
