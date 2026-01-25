package cluster

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/basebandit/kai"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// Ingress represents a Kubernetes Ingress resource.
type Ingress struct {
	Name             string
	Namespace        string
	IngressClassName string
	Labels           map[string]interface{}
	Annotations      map[string]interface{}
	Rules            []kai.IngressRule
	TLS              []kai.IngressTLS
	DefaultBackend   *kai.IngressBackend
}

// Create creates a new Ingress in the specified namespace.
func (i *Ingress) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if err := i.validate(); err != nil {
		return result, err
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, i.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %w", i.Namespace, err)
	}

	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.Name,
			Namespace: i.Namespace,
		},
		Spec: networkingv1.IngressSpec{},
	}

	if i.Labels != nil {
		ingress.ObjectMeta.Labels = convertToStringMap(i.Labels)
	}

	if i.Annotations != nil {
		ingress.ObjectMeta.Annotations = convertToStringMap(i.Annotations)
	}

	if i.IngressClassName != "" {
		ingress.Spec.IngressClassName = &i.IngressClassName
	}

	// Set default backend if specified
	if i.DefaultBackend != nil {
		backend, err := i.createIngressBackend(i.DefaultBackend)
		if err != nil {
			return result, err
		}
		ingress.Spec.DefaultBackend = backend
	}

	// Set rules
	if len(i.Rules) > 0 {
		rules := make([]networkingv1.IngressRule, 0, len(i.Rules))
		for _, rule := range i.Rules {
			ingressRule := networkingv1.IngressRule{
				Host: rule.Host,
			}

			if len(rule.Paths) > 0 {
				paths := make([]networkingv1.HTTPIngressPath, 0, len(rule.Paths))
				for _, path := range rule.Paths {
					pathType := networkingv1.PathTypePrefix
					if path.PathType != "" {
						switch path.PathType {
						case "Exact":
							pathType = networkingv1.PathTypeExact
						case "Prefix":
							pathType = networkingv1.PathTypePrefix
						case "ImplementationSpecific":
							pathType = networkingv1.PathTypeImplementationSpecific
						default:
							return result, fmt.Errorf("invalid path type: %s", path.PathType)
						}
					}

					backend, err := i.createIngressBackend(&kai.IngressBackend{
						ServiceName: path.ServiceName,
						ServicePort: path.ServicePort,
					})
					if err != nil {
						return result, err
					}

					ingressPath := networkingv1.HTTPIngressPath{
						Path:     path.Path,
						PathType: &pathType,
						Backend:  *backend,
					}
					paths = append(paths, ingressPath)
				}
				ingressRule.HTTP = &networkingv1.HTTPIngressRuleValue{
					Paths: paths,
				}
			}

			rules = append(rules, ingressRule)
		}
		ingress.Spec.Rules = rules
	}

	// Set TLS configuration
	if len(i.TLS) > 0 {
		tlsConfigs := make([]networkingv1.IngressTLS, 0, len(i.TLS))
		for _, tls := range i.TLS {
			tlsConfig := networkingv1.IngressTLS{
				Hosts:      tls.Hosts,
				SecretName: tls.SecretName,
			}
			tlsConfigs = append(tlsConfigs, tlsConfig)
		}
		ingress.Spec.TLS = tlsConfigs
	}

	createdIngress, err := client.NetworkingV1().Ingresses(i.Namespace).Create(timeoutCtx, ingress, metav1.CreateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to create Ingress: %w", err)
	}

	result = fmt.Sprintf("Ingress %q created successfully in namespace %q", createdIngress.Name, createdIngress.Namespace)
	if i.IngressClassName != "" {
		result += fmt.Sprintf(" (Class: %s)", i.IngressClassName)
	}

	return result, nil
}

// Get retrieves an Ingress by name from the specified namespace.
func (i *Ingress) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	var ingress *networkingv1.Ingress
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		return !strings.Contains(err.Error(), "not found")
	}, func() error {
		var getErr error
		ingress, getErr = client.NetworkingV1().Ingresses(i.Namespace).Get(ctx, i.Name, metav1.GetOptions{})
		return getErr
	})

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, fmt.Errorf("Ingress %q not found in namespace %q", i.Name, i.Namespace)
		}
		return result, fmt.Errorf("failed to get Ingress %q: %v", i.Name, err)
	}

	return formatIngress(ingress), nil
}

// List retrieves all Ingresses matching the specified criteria.
func (i *Ingress) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
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

	var ingresses *networkingv1.IngressList
	if allNamespaces {
		ingresses, err = client.NetworkingV1().Ingresses("").List(timeoutCtx, listOptions)
	} else {
		ingresses, err = client.NetworkingV1().Ingresses(i.Namespace).List(timeoutCtx, listOptions)
	}

	if err != nil {
		return result, fmt.Errorf("failed to list Ingresses: %w", err)
	}

	if len(ingresses.Items) == 0 {
		if labelSelector != "" {
			return result, errors.New("no Ingresses found matching the specified label selector")
		}
		if allNamespaces {
			return result, errors.New("no Ingresses found in any namespace")
		}
		return result, fmt.Errorf("no Ingresses found in namespace %q", i.Namespace)
	}

	return formatIngressList(ingresses, allNamespaces), nil
}

// Update updates an existing Ingress in the specified namespace.
func (i *Ingress) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if i.Name == "" {
		return result, errors.New("Ingress name is required for update")
	}

	if i.Namespace == "" {
		return result, errors.New("namespace is required for update")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// Get the existing ingress
	existingIngress, err := client.NetworkingV1().Ingresses(i.Namespace).Get(timeoutCtx, i.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("Ingress %q not found in namespace %q: %w", i.Name, i.Namespace, err)
	}

	// Update fields if specified
	if i.IngressClassName != "" {
		existingIngress.Spec.IngressClassName = &i.IngressClassName
	}

	if i.Labels != nil {
		if existingIngress.Labels == nil {
			existingIngress.Labels = make(map[string]string)
		}
		for k, v := range convertToStringMap(i.Labels) {
			existingIngress.Labels[k] = v
		}
	}

	if i.Annotations != nil {
		if existingIngress.Annotations == nil {
			existingIngress.Annotations = make(map[string]string)
		}
		for k, v := range convertToStringMap(i.Annotations) {
			existingIngress.Annotations[k] = v
		}
	}

	// Update rules if specified
	if len(i.Rules) > 0 {
		rules := make([]networkingv1.IngressRule, 0, len(i.Rules))
		for _, rule := range i.Rules {
			ingressRule := networkingv1.IngressRule{
				Host: rule.Host,
			}

			if len(rule.Paths) > 0 {
				paths := make([]networkingv1.HTTPIngressPath, 0, len(rule.Paths))
				for _, path := range rule.Paths {
					pathType := networkingv1.PathTypePrefix
					if path.PathType != "" {
						switch path.PathType {
						case "Exact":
							pathType = networkingv1.PathTypeExact
						case "Prefix":
							pathType = networkingv1.PathTypePrefix
						case "ImplementationSpecific":
							pathType = networkingv1.PathTypeImplementationSpecific
						}
					}

					backend, err := i.createIngressBackend(&kai.IngressBackend{
						ServiceName: path.ServiceName,
						ServicePort: path.ServicePort,
					})
					if err != nil {
						return result, err
					}

					ingressPath := networkingv1.HTTPIngressPath{
						Path:     path.Path,
						PathType: &pathType,
						Backend:  *backend,
					}
					paths = append(paths, ingressPath)
				}
				ingressRule.HTTP = &networkingv1.HTTPIngressRuleValue{
					Paths: paths,
				}
			}

			rules = append(rules, ingressRule)
		}
		existingIngress.Spec.Rules = rules
	}

	// Update TLS if specified
	if len(i.TLS) > 0 {
		tlsConfigs := make([]networkingv1.IngressTLS, 0, len(i.TLS))
		for _, tls := range i.TLS {
			tlsConfig := networkingv1.IngressTLS{
				Hosts:      tls.Hosts,
				SecretName: tls.SecretName,
			}
			tlsConfigs = append(tlsConfigs, tlsConfig)
		}
		existingIngress.Spec.TLS = tlsConfigs
	}

	updatedIngress, err := client.NetworkingV1().Ingresses(i.Namespace).Update(timeoutCtx, existingIngress, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to update Ingress: %w", err)
	}

	result = fmt.Sprintf("Ingress %q updated successfully in namespace %q", updatedIngress.Name, updatedIngress.Namespace)
	return result, nil
}

// Delete removes an Ingress by name from the specified namespace.
func (i *Ingress) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if i.Name == "" {
		return result, errors.New("Ingress name is required for deletion")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err = client.NetworkingV1().Ingresses(i.Namespace).Get(timeoutCtx, i.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("Ingress %q not found in namespace %q: %w", i.Name, i.Namespace, err)
	}

	err = client.NetworkingV1().Ingresses(i.Namespace).Delete(timeoutCtx, i.Name, metav1.DeleteOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to delete Ingress %q: %w", i.Name, err)
	}

	result = fmt.Sprintf("Ingress %q deleted successfully from namespace %q", i.Name, i.Namespace)
	return result, nil
}

func (i *Ingress) validate() error {
	if i.Name == "" {
		return errors.New("Ingress name is required")
	}
	if i.Namespace == "" {
		return errors.New("namespace is required")
	}

	// Validate rules
	for idx, rule := range i.Rules {
		for pathIdx, path := range rule.Paths {
			if path.ServiceName == "" {
				return fmt.Errorf("rule %d, path %d: service name is required", idx, pathIdx)
			}
			if path.ServicePort == nil {
				return fmt.Errorf("rule %d, path %d: service port is required", idx, pathIdx)
			}
		}
	}

	return nil
}

func (i *Ingress) createIngressBackend(backend *kai.IngressBackend) (*networkingv1.IngressBackend, error) {
	if backend.ServiceName == "" {
		return nil, errors.New("service name is required for backend")
	}

	ingressBackend := &networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: backend.ServiceName,
		},
	}

	switch v := backend.ServicePort.(type) {
	case int:
		ingressBackend.Service.Port = networkingv1.ServiceBackendPort{
			Number: int32(v),
		}
	case int32:
		ingressBackend.Service.Port = networkingv1.ServiceBackendPort{
			Number: v,
		}
	case float64:
		ingressBackend.Service.Port = networkingv1.ServiceBackendPort{
			Number: int32(v),
		}
	case string:
		ingressBackend.Service.Port = networkingv1.ServiceBackendPort{
			Name: v,
		}
	default:
		return nil, fmt.Errorf("unsupported service port type: %T", v)
	}

	return ingressBackend, nil
}
