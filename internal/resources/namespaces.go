package resources

import (
	"context"
	"fmt"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// NamespaceExtra holds the complete namespace state in a serializable form for GOB encoding
type NamespaceExtra struct {
	Labels        []string
	Annotations   []string
	Status        string
	Finalizers    []string
	ResourceQuota []string
	LimitRange    []string
	Age           serializable.Time
}

// Copy performs a deep copy of NamespaceExtra
func (p NamespaceExtra) Copy() Copyable {
	return NamespaceExtra{
		Labels:        misc.DeepCopyArray(p.Labels),
		Annotations:   misc.DeepCopyArray(p.Annotations),
		Status:        p.Status,
		Finalizers:    misc.DeepCopyArray(p.Finalizers),
		ResourceQuota: misc.DeepCopyArray(p.ResourceQuota),
		LimitRange:    misc.DeepCopyArray(p.LimitRange),
		Age:           p.Age,
	}
}

// newNamespaceExtra constructs a NamespaceExtra from a Kubernetes Namespace
func newNamespaceExtra(ns *corev1.Namespace) NamespaceExtra {
	if ns == nil {
		return NamespaceExtra{}
	}

	// Convert FinalizerName slice to string slice
	finalizers := make([]string, len(ns.Spec.Finalizers))
	for i, f := range ns.Spec.Finalizers {
		finalizers[i] = string(f)
	}

	return NamespaceExtra{
		Labels:        misc.RenderMapOfStrings(ns.Labels),
		Annotations:   misc.RenderMapOfStrings(ns.Annotations),
		Status:        string(ns.Status.Phase),
		Finalizers:    finalizers,
		ResourceQuota: []string{}, // Populated by external query
		LimitRange:    []string{}, // Populated by external query
		Age:           serializable.NewTime(ns.CreationTimestamp.Time),
	}
}

type NamespaceRenderer struct{}

func renderNamespaceExtra(extra NamespaceExtra) []string {
	output := []string{
		fmt.Sprintf("Status:       %s", extra.Status),
	}

	// Add Labels section
	output = append(output, fmt.Sprintf("Labels:       %s", misc.FormatNilArray(extra.Labels)))

	// Add Annotations section
	output = append(output, fmt.Sprintf("Annotations:  %s", misc.FormatNilArray(extra.Annotations)))

	// Add Finalizers if present
	if len(extra.Finalizers) > 0 {
		output = append(output, "Finalizers:")
		for _, finalizer := range extra.Finalizers {
			output = append(output, fmt.Sprintf("                %s", finalizer))
		}
	}

	// Add Resource Quota section
	output = append(output, "\nResource Quota:")
	if len(extra.ResourceQuota) == 0 {
		output = append(output, "No resource quota.")
	} else {
		for _, quota := range extra.ResourceQuota {
			output = append(output, fmt.Sprintf("  %s", quota))
		}
	}

	// Add Limit Range section
	output = append(output, "\nLimit Range:")
	if len(extra.LimitRange) == 0 {
		output = append(output, "No LimitRange resource.")
	} else {
		for _, limit := range extra.LimitRange {
			output = append(output, fmt.Sprintf("  %s", limit))
		}
	}

	return output
}

func (r NamespaceRenderer) Render(resource Resource, details bool) []string {
	extra, ok := resource.Extra.(NamespaceExtra)
	if !ok {
		return []string{"Error: Invalid extra type"}
	}

	if details {
		return renderNamespaceExtra(extra)
	}
	return []string{resource.Key()}
}

type NamespaceWatcher struct{}

func (n NamespaceWatcher) Tick()                      {}
func (n NamespaceWatcher) Kind() string               { return "Namespace" }
func (n NamespaceWatcher) Renderer() ResourceRenderer { return NamespaceRenderer{} }

func (n NamespaceWatcher) convert(obj runtime.Object) *corev1.Namespace {
	if obj == nil {
		return nil
	}
	ret, ok := obj.(*corev1.Namespace)
	if !ok {
		return nil
	}
	return ret
}

func (n NamespaceWatcher) ToResource(obj runtime.Object) Resource {
	ns := n.convert(obj)
	if ns == nil {
		return Resource{}
	}
	extra := newNamespaceExtra(ns)
	return NewK8sResource(n.Kind(), ns, extra)
}

func watchForNamespace(watcher *K8sWatcher, k conn.KhronosConn) error {
	watchChan, err := k.Client.CoreV1().Namespaces().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to watch namespaces: %w", err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), NamespaceWatcher{})
	return nil
}
