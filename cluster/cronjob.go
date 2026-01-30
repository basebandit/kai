package cluster

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/basebandit/kai"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// CronJob represents a Kubernetes CronJob resource.
type CronJob struct {
	Name                       string
	Namespace                  string
	Schedule                   string
	Image                      string
	Command                    []interface{}
	Args                       []interface{}
	RestartPolicy              string
	ConcurrencyPolicy          string
	Suspend                    *bool
	SuccessfulJobsHistoryLimit *int32
	FailedJobsHistoryLimit     *int32
	StartingDeadlineSeconds    *int64
	BackoffLimit               *int32
	Labels                     map[string]interface{}
	Env                        map[string]interface{}
	ImagePullPolicy            string
	ImagePullSecrets           []interface{}
}

// Create creates a new CronJob in the specified namespace.
func (c *CronJob) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if err := c.validate(); err != nil {
		slog.Warn("invalid CronJob input",
			slog.String("name", c.Name),
			slog.String("namespace", c.Namespace),
			slog.String("error", err.Error()),
		)
		return result, err
	}

	slog.Debug("CronJob create requested",
		slog.String("name", c.Name),
		slog.String("namespace", c.Namespace),
	)

	client, err := cm.GetCurrentClient()
	if err != nil {
		slog.Warn("failed to get client for CronJob create",
			slog.String("name", c.Name),
			slog.String("namespace", c.Namespace),
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, c.Namespace, metav1.GetOptions{})
	if err != nil {
		slog.Warn("namespace not found for CronJob create",
			slog.String("name", c.Name),
			slog.String("namespace", c.Namespace),
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("namespace %q not found: %w", c.Namespace, err)
	}

	restartPolicy := corev1.RestartPolicyOnFailure
	if c.RestartPolicy != "" {
		restartPolicy = corev1.RestartPolicy(c.RestartPolicy)
	}

	podSpec := corev1.PodSpec{
		RestartPolicy: restartPolicy,
		Containers: []corev1.Container{
			{
				Name:  c.Name,
				Image: c.Image,
			},
		},
	}

	if len(c.Command) > 0 {
		podSpec.Containers[0].Command = convertToStringSlice(c.Command)
	}

	if len(c.Args) > 0 {
		podSpec.Containers[0].Args = convertToStringSlice(c.Args)
	}

	if c.Env != nil {
		podSpec.Containers[0].Env = convertToEnvVars(c.Env)
	}

	if c.ImagePullPolicy != "" {
		podSpec.Containers[0].ImagePullPolicy = corev1.PullPolicy(c.ImagePullPolicy)
	}

	if len(c.ImagePullSecrets) > 0 {
		podSpec.ImagePullSecrets = convertToLocalObjectReferences(c.ImagePullSecrets)
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: c.Schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: podSpec,
					},
				},
			},
		},
	}

	if c.Labels != nil {
		labels := convertToStringMap(c.Labels)
		if len(labels) > 0 {
			cronJob.ObjectMeta.Labels = labels
			cronJob.Spec.JobTemplate.ObjectMeta.Labels = labels
			cronJob.Spec.JobTemplate.Spec.Template.ObjectMeta.Labels = labels
		}
	}

	if c.ConcurrencyPolicy != "" {
		cronJob.Spec.ConcurrencyPolicy = batchv1.ConcurrencyPolicy(c.ConcurrencyPolicy)
	}

	if c.Suspend != nil {
		cronJob.Spec.Suspend = c.Suspend
	}

	if c.SuccessfulJobsHistoryLimit != nil {
		cronJob.Spec.SuccessfulJobsHistoryLimit = c.SuccessfulJobsHistoryLimit
	}

	if c.FailedJobsHistoryLimit != nil {
		cronJob.Spec.FailedJobsHistoryLimit = c.FailedJobsHistoryLimit
	}

	if c.StartingDeadlineSeconds != nil {
		cronJob.Spec.StartingDeadlineSeconds = c.StartingDeadlineSeconds
	}

	if c.BackoffLimit != nil {
		cronJob.Spec.JobTemplate.Spec.BackoffLimit = c.BackoffLimit
	}

	createdCronJob, err := client.BatchV1().CronJobs(c.Namespace).Create(timeoutCtx, cronJob, metav1.CreateOptions{})
	if err != nil {
		slog.Warn("failed to create CronJob",
			slog.String("name", c.Name),
			slog.String("namespace", c.Namespace),
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("failed to create CronJob: %w", err)
	}

	slog.Info("CronJob created",
		slog.String("name", createdCronJob.Name),
		slog.String("namespace", createdCronJob.Namespace),
		slog.String("schedule", createdCronJob.Spec.Schedule),
	)

	result = fmt.Sprintf("CronJob %q created successfully in namespace %q with schedule %q", createdCronJob.Name, createdCronJob.Namespace, createdCronJob.Spec.Schedule)
	return result, nil
}

// Get retrieves a CronJob by name from the specified namespace.
func (c *CronJob) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	slog.Debug("CronJob get requested",
		slog.String("name", c.Name),
		slog.String("namespace", c.Namespace),
	)

	client, err := cm.GetCurrentClient()
	if err != nil {
		slog.Warn("failed to get client for CronJob get",
			slog.String("name", c.Name),
			slog.String("namespace", c.Namespace),
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("error getting client: %w", err)
	}

	var cronJob *batchv1.CronJob
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		return !strings.Contains(err.Error(), "not found")
	}, func() error {
		var getErr error
		cronJob, getErr = client.BatchV1().CronJobs(c.Namespace).Get(ctx, c.Name, metav1.GetOptions{})
		return getErr
	})

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			slog.Warn("CronJob not found",
				slog.String("name", c.Name),
				slog.String("namespace", c.Namespace),
				slog.String("error", err.Error()),
			)
			return result, fmt.Errorf("CronJob %q not found in namespace %q", c.Name, c.Namespace)
		}
		slog.Warn("failed to get CronJob",
			slog.String("name", c.Name),
			slog.String("namespace", c.Namespace),
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("failed to get CronJob %q: %v", c.Name, err)
	}

	return formatCronJob(cronJob), nil
}

// List retrieves all CronJobs matching the specified criteria.
func (c *CronJob) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
	var result string

	slog.Debug("CronJob list requested",
		slog.Bool("all_namespaces", allNamespaces),
		slog.String("namespace", c.Namespace),
		slog.String("label_selector", labelSelector),
	)

	client, err := cm.GetCurrentClient()
	if err != nil {
		slog.Warn("failed to get client for CronJob list",
			slog.Bool("all_namespaces", allNamespaces),
			slog.String("namespace", c.Namespace),
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("error getting client: %w", err)
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	var cronJobs *batchv1.CronJobList
	if allNamespaces {
		cronJobs, err = client.BatchV1().CronJobs("").List(timeoutCtx, listOptions)
	} else {
		cronJobs, err = client.BatchV1().CronJobs(c.Namespace).List(timeoutCtx, listOptions)
	}

	if err != nil {
		slog.Warn("failed to list CronJobs",
			slog.Bool("all_namespaces", allNamespaces),
			slog.String("namespace", c.Namespace),
			slog.String("label_selector", labelSelector),
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("failed to list CronJobs: %w", err)
	}

	if len(cronJobs.Items) == 0 {
		if labelSelector != "" {
			return result, errors.New("no CronJobs found matching the specified label selector")
		}
		if allNamespaces {
			return result, errors.New("no CronJobs found in any namespace")
		}
		return result, fmt.Errorf("no CronJobs found in namespace %q", c.Namespace)
	}

	return formatCronJobList(cronJobs, allNamespaces), nil
}

// Delete removes a CronJob by name from the specified namespace.
func (c *CronJob) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if c.Name == "" {
		slog.Warn("CronJob delete missing name",
			slog.String("namespace", c.Namespace),
		)
		return result, errors.New("CronJob name is required for deletion")
	}

	slog.Debug("CronJob delete requested",
		slog.String("name", c.Name),
		slog.String("namespace", c.Namespace),
	)

	client, err := cm.GetCurrentClient()
	if err != nil {
		slog.Warn("failed to get client for CronJob delete",
			slog.String("name", c.Name),
			slog.String("namespace", c.Namespace),
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err = client.BatchV1().CronJobs(c.Namespace).Get(timeoutCtx, c.Name, metav1.GetOptions{})
	if err != nil {
		slog.Warn("CronJob not found for delete",
			slog.String("name", c.Name),
			slog.String("namespace", c.Namespace),
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("CronJob %q not found in namespace %q: %w", c.Name, c.Namespace, err)
	}

	propagationPolicy := metav1.DeletePropagationBackground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	}

	err = client.BatchV1().CronJobs(c.Namespace).Delete(timeoutCtx, c.Name, deleteOptions)
	if err != nil {
		slog.Warn("failed to delete CronJob",
			slog.String("name", c.Name),
			slog.String("namespace", c.Namespace),
			slog.String("error", err.Error()),
		)
		return result, fmt.Errorf("failed to delete CronJob %q: %w", c.Name, err)
	}

	slog.Info("CronJob deleted",
		slog.String("name", c.Name),
		slog.String("namespace", c.Namespace),
	)

	result = fmt.Sprintf("CronJob %q deleted successfully from namespace %q", c.Name, c.Namespace)
	return result, nil
}

// Update updates mutable fields of an existing CronJob
func (c *CronJob) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if c.Name == "" {
		return result, errors.New("CronJob name is required")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	cronJob, err := client.BatchV1().CronJobs(c.Namespace).Get(timeoutCtx, c.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get CronJob: %w", err)
	}

	if len(c.Labels) > 0 {
		if cronJob.Labels == nil {
			cronJob.Labels = make(map[string]string)
		}
		for k, v := range convertToStringMap(c.Labels) {
			cronJob.Labels[k] = v
		}
	}

	if c.Schedule != "" {
		cronJob.Spec.Schedule = c.Schedule
	}

	if c.ConcurrencyPolicy != "" {
		cronJob.Spec.ConcurrencyPolicy = batchv1.ConcurrencyPolicy(c.ConcurrencyPolicy)
	}

	if c.SuccessfulJobsHistoryLimit != nil {
		cronJob.Spec.SuccessfulJobsHistoryLimit = c.SuccessfulJobsHistoryLimit
	}

	if c.FailedJobsHistoryLimit != nil {
		cronJob.Spec.FailedJobsHistoryLimit = c.FailedJobsHistoryLimit
	}

	updatedCronJob, err := client.BatchV1().CronJobs(c.Namespace).Update(timeoutCtx, cronJob, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to update CronJob: %w", err)
	}

	result = fmt.Sprintf("CronJob %q updated successfully in namespace %q", updatedCronJob.Name, updatedCronJob.Namespace)
	return result, nil
}

// SetSuspended sets the suspend state of a CronJob
func (c *CronJob) SetSuspended(ctx context.Context, cm kai.ClusterManager, suspend bool) (string, error) {
	var result string

	if c.Name == "" {
		return result, errors.New("CronJob name is required")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	cronJob, err := client.BatchV1().CronJobs(c.Namespace).Get(timeoutCtx, c.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get CronJob: %w", err)
	}

	cronJob.Spec.Suspend = &suspend

	_, err = client.BatchV1().CronJobs(c.Namespace).Update(timeoutCtx, cronJob, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to set suspend state for CronJob: %w", err)
	}

	if suspend {
		result = fmt.Sprintf("CronJob %q suspended in namespace %q", c.Name, c.Namespace)
	} else {
		result = fmt.Sprintf("CronJob %q resumed in namespace %q", c.Name, c.Namespace)
	}
	return result, nil
}

func (c *CronJob) validate() error {
	if c.Name == "" {
		return errors.New("CronJob name is required")
	}
	if c.Namespace == "" {
		return errors.New("namespace is required")
	}
	if c.Schedule == "" {
		return errors.New("schedule is required")
	}
	if c.Image == "" {
		return errors.New("image is required")
	}
	return nil
}
