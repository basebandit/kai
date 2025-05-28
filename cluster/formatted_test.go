package cluster

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{45 * time.Second, "45s"},
		{5 * time.Minute, "5m"},
		{3 * time.Hour, "3h"},
		{2 * 24 * time.Hour, "2d"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		assert.Equal(t, tt.expected, result)
	}
}

func TestConvertToStringMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: map[string]string{},
		},
		{
			name: "mixed types",
			input: map[string]interface{}{
				"string": "value",
				"int":    123,
				"bool":   true,
			},
			expected: map[string]string{
				"string": "value",
				"int":    "123",
				"bool":   "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToStringMap(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, len(tt.expected), len(result))
				for k, v := range tt.expected {
					assert.Equal(t, v, result[k])
				}
			}
		})
	}
}

func TestFormatNamespace(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-ns",
			CreationTimestamp: metav1.Time{Time: time.Now()},
			Labels:            map[string]string{"env": "test"},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	result := formatNamespace(ns)
	assert.Contains(t, result, "Namespace: test-ns")
	assert.Contains(t, result, "Status: Active")
	assert.Contains(t, result, "- env: test")
}

func TestFormatNamespaceList(t *testing.T) {
	nsList := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "default",
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-24 * time.Hour)},
				},
				Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
			},
		},
	}

	result := formatNamespaceList(nsList, "")
	assert.Contains(t, result, "Namespaces:")
	assert.Contains(t, result, "• default: Status=Active, Age=1d")
	assert.Contains(t, result, "Total: 1 namespace(s)")

	resultWithSelector := formatNamespaceList(nsList, "env=prod")
	assert.Contains(t, resultWithSelector, "matching label selector 'env=prod'")
}

func TestFormatNamespaceListEmptyStatus(t *testing.T) {
	nsList := &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test",
					CreationTimestamp: metav1.Time{Time: time.Now()},
				},
				Status: corev1.NamespaceStatus{Phase: ""},
			},
		},
	}

	result := formatNamespaceList(nsList, "")
	assert.Contains(t, result, "Status=Active")
}
