package cluster

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/basebandit/kai"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// Job represents a Kubernetes Job resource.
type Job struct {
	Name             string
	Namespace        string
	Image            string
	Command          []interface{}
	Args             []interface{}
	RestartPolicy    string
	BackoffLimit     *int32
	Completions      *int32
	Parallelism      *int32
	Labels           map[string]interface{}
	Env              map[string]interface{}
	ImagePullPolicy  string
	ImagePullSecrets []interface{}
}

// Create creates a new Job in the specified namespace.
func (j *Job) Create(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if err := j.validate(); err != nil {
		return result, err
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err = client.CoreV1().Namespaces().Get(timeoutCtx, j.Namespace, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("namespace %q not found: %w", j.Namespace, err)
	}

	restartPolicy := corev1.RestartPolicyNever
	if j.RestartPolicy != "" {
		restartPolicy = corev1.RestartPolicy(j.RestartPolicy)
	}

	podSpec := corev1.PodSpec{
		RestartPolicy: restartPolicy,
		Containers: []corev1.Container{
			{
				Name:  j.Name,
				Image: j.Image,
			},
		},
	}

	if len(j.Command) > 0 {
		podSpec.Containers[0].Command = convertToStringSlice(j.Command)
	}

	if len(j.Args) > 0 {
		podSpec.Containers[0].Args = convertToStringSlice(j.Args)
	}

	if j.Env != nil {
		podSpec.Containers[0].Env = convertToEnvVars(j.Env)
	}

	if j.ImagePullPolicy != "" {
		podSpec.Containers[0].ImagePullPolicy = corev1.PullPolicy(j.ImagePullPolicy)
	}

	if len(j.ImagePullSecrets) > 0 {
		podSpec.ImagePullSecrets = convertToLocalObjectReferences(j.ImagePullSecrets)
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.Name,
			Namespace: j.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}

	if j.Labels != nil {
		labels := convertToStringMap(j.Labels)
		if len(labels) > 0 {
			job.ObjectMeta.Labels = labels
			job.Spec.Template.ObjectMeta.Labels = labels
		}
	}

	if j.BackoffLimit != nil {
		job.Spec.BackoffLimit = j.BackoffLimit
	}

	if j.Completions != nil {
		job.Spec.Completions = j.Completions
	}

	if j.Parallelism != nil {
		job.Spec.Parallelism = j.Parallelism
	}

	createdJob, err := client.BatchV1().Jobs(j.Namespace).Create(timeoutCtx, job, metav1.CreateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to create Job: %w", err)
	}

	result = fmt.Sprintf("Job %q created successfully in namespace %q", createdJob.Name, createdJob.Namespace)
	return result, nil
}

// Get retrieves a Job by name from the specified namespace.
func (j *Job) Get(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	var job *batchv1.Job
	err = retry.OnError(retry.DefaultRetry, func(err error) bool {
		return !strings.Contains(err.Error(), "not found")
	}, func() error {
		var getErr error
		job, getErr = client.BatchV1().Jobs(j.Namespace).Get(ctx, j.Name, metav1.GetOptions{})
		return getErr
	})

	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return result, fmt.Errorf("Job %q not found in namespace %q", j.Name, j.Namespace)
		}
		return result, fmt.Errorf("failed to get Job %q: %v", j.Name, err)
	}

	return formatJob(job), nil
}

// List retrieves all Jobs matching the specified criteria.
func (j *Job) List(ctx context.Context, cm kai.ClusterManager, allNamespaces bool, labelSelector string) (string, error) {
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

	var jobs *batchv1.JobList
	if allNamespaces {
		jobs, err = client.BatchV1().Jobs("").List(timeoutCtx, listOptions)
	} else {
		jobs, err = client.BatchV1().Jobs(j.Namespace).List(timeoutCtx, listOptions)
	}

	if err != nil {
		return result, fmt.Errorf("failed to list Jobs: %w", err)
	}

	if len(jobs.Items) == 0 {
		if labelSelector != "" {
			return result, errors.New("no Jobs found matching the specified label selector")
		}
		if allNamespaces {
			return result, errors.New("no Jobs found in any namespace")
		}
		return result, fmt.Errorf("no Jobs found in namespace %q", j.Namespace)
	}

	return formatJobList(jobs, allNamespaces), nil
}

// Delete removes a Job by name from the specified namespace.
func (j *Job) Delete(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if j.Name == "" {
		return result, errors.New("Job name is required for deletion")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	_, err = client.BatchV1().Jobs(j.Namespace).Get(timeoutCtx, j.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("Job %q not found in namespace %q: %w", j.Name, j.Namespace, err)
	}

	propagationPolicy := metav1.DeletePropagationBackground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	}

	err = client.BatchV1().Jobs(j.Namespace).Delete(timeoutCtx, j.Name, deleteOptions)
	if err != nil {
		return result, fmt.Errorf("failed to delete Job %q: %w", j.Name, err)
	}

	result = fmt.Sprintf("Job %q deleted successfully from namespace %q", j.Name, j.Namespace)
	return result, nil
}

// Update updates mutable fields of an existing Job
func (j *Job) Update(ctx context.Context, cm kai.ClusterManager) (string, error) {
	var result string

	if j.Name == "" {
		return result, errors.New("Job name is required")
	}

	client, err := cm.GetCurrentClient()
	if err != nil {
		return result, fmt.Errorf("error getting client: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	job, err := client.BatchV1().Jobs(j.Namespace).Get(timeoutCtx, j.Name, metav1.GetOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to get Job: %w", err)
	}

	if len(j.Labels) > 0 {
		if job.Labels == nil {
			job.Labels = make(map[string]string)
		}
		for k, v := range convertToStringMap(j.Labels) {
			job.Labels[k] = v
		}
	}

	if j.Parallelism != nil {
		job.Spec.Parallelism = j.Parallelism
	}

	updatedJob, err := client.BatchV1().Jobs(j.Namespace).Update(timeoutCtx, job, metav1.UpdateOptions{})
	if err != nil {
		return result, fmt.Errorf("failed to update Job: %w", err)
	}

	result = fmt.Sprintf("Job %q updated successfully in namespace %q", updatedJob.Name, updatedJob.Namespace)
	return result, nil
}

func (j *Job) validate() error {
	if j.Name == "" {
		return errors.New("Job name is required")
	}
	if j.Namespace == "" {
		return errors.New("namespace is required")
	}
	if j.Image == "" {
		return errors.New("image is required")
	}
	return nil
}
