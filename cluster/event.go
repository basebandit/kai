package cluster

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/basebandit/kai"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

// Event represents a query for Kubernetes events.
type Event struct {
	Namespace      string
	AllNamespaces  bool
	Type           string // "Warning" or "Normal"; empty means all types
	InvolvedObject string // filter to a single involved object by name
	Limit          int64
}

// List returns events for the requested scope, most recent first.
func (e *Event) List(ctx context.Context, cm kai.ClusterManager) (string, error) {
	client, err := cm.GetCurrentClient()
	if err != nil {
		return "", fmt.Errorf("error getting client: %w", err)
	}

	namespace := ""
	if !e.AllNamespaces {
		namespace = e.Namespace
		if namespace == "" {
			namespace = cm.GetCurrentNamespace()
		}
	}

	listOptions := metav1.ListOptions{}
	if e.Limit > 0 {
		listOptions.Limit = e.Limit
	}

	var selectors []fields.Selector
	if e.Type != "" {
		selectors = append(selectors, fields.OneTermEqualSelector("type", e.Type))
	}
	if e.InvolvedObject != "" {
		selectors = append(selectors, fields.OneTermEqualSelector("involvedObject.name", e.InvolvedObject))
	}
	if len(selectors) > 0 {
		listOptions.FieldSelector = fields.AndSelectors(selectors...).String()
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, listTimeout)
	defer cancel()

	events, err := client.CoreV1().Events(namespace).List(timeoutCtx, listOptions)
	if err != nil {
		return "", fmt.Errorf("failed to list events: %w", err)
	}

	if len(events.Items) == 0 {
		return "No events found", nil
	}

	return formatEventList(events, e.AllNamespaces), nil
}

func eventTime(e corev1.Event) metav1.Time {
	if !e.LastTimestamp.IsZero() {
		return e.LastTimestamp
	}
	if !e.EventTime.IsZero() {
		return metav1.Time{Time: e.EventTime.Time}
	}
	return e.FirstTimestamp
}

func formatEventList(events *corev1.EventList, allNamespaces bool) string {
	items := make([]corev1.Event, len(events.Items))
	copy(items, events.Items)
	sort.Slice(items, func(i, j int) bool {
		return eventTime(items[i]).After(eventTime(items[j]).Time)
	})

	var sb strings.Builder
	fmt.Fprintf(&sb, "Events (%d):\n", len(items))
	for _, ev := range items {
		obj := ev.InvolvedObject.Kind
		if ev.InvolvedObject.Name != "" {
			obj = fmt.Sprintf("%s/%s", ev.InvolvedObject.Kind, ev.InvolvedObject.Name)
		}
		age := formatDuration(time.Since(eventTime(ev).Time))
		line := fmt.Sprintf("• [%s] %s", ev.Type, ev.Reason)
		if allNamespaces {
			line += fmt.Sprintf(" (ns: %s)", ev.Namespace)
		}
		sb.WriteString(line + "\n")
		fmt.Fprintf(&sb, "    object: %s\n", obj)
		if ev.Count > 1 {
			fmt.Fprintf(&sb, "    count: %d, last seen: %s ago\n", ev.Count, age)
		} else {
			fmt.Fprintf(&sb, "    last seen: %s ago\n", age)
		}
		fmt.Fprintf(&sb, "    message: %s\n", strings.TrimSpace(ev.Message))
	}
	return strings.TrimRight(sb.String(), "\n")
}
