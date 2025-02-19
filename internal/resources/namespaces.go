package resources

import (
	"context"
	"fmt"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
)

// NamespaceExtra holds all necessary namespace state
type NamespaceExtra struct {
	Name              string
	UID               string
	CreationTimestamp string
	Labels            []string
	Annotations       []string
	Status            string
}

// Copy creates a deep copy of NamespaceExtra
func (p NamespaceExtra) Copy() Copyable {
	return NamespaceExtra{
		Name:              p.Name,
		UID:               p.UID,
		CreationTimestamp: p.CreationTimestamp,
		Labels:            misc.DeepCopyArray(p.Labels),
		Annotations:       misc.DeepCopyArray(p.Annotations),
		Status:            p.Status,
	}
}

// newNamespaceExtra constructs a NamespaceExtra from a *corev1.Namespace
func newNamespaceExtra(ns *corev1.Namespace) NamespaceExtra {

	return NamespaceExtra{
		Name:              ns.Name,
		UID:               string(ns.UID),
		CreationTimestamp: duration.HumanDuration(v1.Now().Sub(ns.CreationTimestamp.Time)),
		Labels:            misc.RenderMapOfStrings("Labels", ns.Labels),
		Annotations:       misc.RenderMapOfStrings("Annotations", ns.Annotations),
		Status:            string(ns.Status.Phase),
	}
}

type NamespaceRenderer struct {
}

// renderNamespaceExtra formats the data similar to `kubectl get namespaces`
func renderNamespaceExtra(extra NamespaceExtra) []string {
	output := []string{
		fmt.Sprintf("Name:          %s", extra.Name),
		fmt.Sprintf("UID:           %s", extra.UID),
		fmt.Sprintf("Created:       %s ago", extra.CreationTimestamp),
		fmt.Sprintf("Status:        %s", extra.Status),
	}

	output = append(output, extra.Labels...)
	output = append(output, extra.Annotations...)

	return output
}

func (r NamespaceRenderer) Render(resource Resource, details bool) []string {
	extra := resource.Extra.(NamespaceExtra)

	if details {
		return renderNamespaceExtra(extra)
	}

	return []string{resource.Key()}
}

type NamespaceWatcher struct {
}

func (n NamespaceWatcher) Tick() {
}

func (n NamespaceWatcher) Kind() string {
	return "Namespace"
}

func (n NamespaceWatcher) Renderer() ResourceRenderer {
	return NamespaceRenderer{}
}

func (n NamespaceWatcher) convert(obj runtime.Object) *corev1.Namespace {
	ret, ok := obj.(*corev1.Namespace)
	if !ok {
		return nil
	}
	return ret
}

func (n NamespaceWatcher) ToResource(obj runtime.Object) Resource {
	ns := n.convert(obj)
	extra := newNamespaceExtra(ns)
	return NewK8sResource(n.Kind(), ns, extra)
}

func watchForNamespace(watcher *K8sWatcher, k conn.KhronosConn) error {
	watchChan, err := k.Client.CoreV1().Namespaces().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), NamespaceWatcher{})

	return nil
}
