package resources

import (
	"context"
	"fmt"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/misc/format"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type NamespacedRenderer struct {
}

func (r NamespacedRenderer) Render(resource Resource, details bool) []string {
	extra := resource.Extra.(NamespaceExtra)

	if details {
		out := []string{}

		out = append(out, fmt.Sprintf("Name: %s", resource.Name))
		out = append(out, fmt.Sprintf("Status: %s", extra.Status))
		out = append(out, extra.Labels...)
		out = append(out, extra.Annotations...)

		return out
	}

	return []string{resource.Name}
}

type NamespaceWatcher struct {
}

func (n NamespaceWatcher) Tick() {
}

func (n NamespaceWatcher) Kind() string {
	return "Namespace"
}

func (n NamespaceWatcher) Renderer() ResourceRenderer {
	return NamespacedRenderer{}
}

func (n NamespaceWatcher) convert(obj runtime.Object) *corev1.Namespace {
	ret, ok := obj.(*corev1.Namespace)
	if !ok {
		return nil
	}
	return ret
}

type NamespaceExtra struct {
	Status      string
	Labels      []string
	Annotations []string
}

func (p NamespaceExtra) Copy() Copyable {
	return NamespaceExtra{
		Status:      p.Status,
		Labels:      misc.DeepCopyArray(p.Labels),
		Annotations: misc.DeepCopyArray(p.Annotations),
	}
}

func (n NamespaceWatcher) ToResource(obj runtime.Object) Resource {
	ns := n.convert(obj)

	extra := NamespaceExtra{
		Status:      ns.Status.String(),
		Labels:      misc.RenderMapOfStrings("Labels:", ns.GetLabels()),
		Annotations: misc.RenderMapOfStrings("Annotations:", ns.GetAnnotations()),
	}

	return NewK8sResource(n.Kind(), ns, format.FormatNamespaceDetails(n.convert(obj)), extra)
}

func watchForNamespaces(watcher *K8sWatcher, k conn.KhronosConn) error {
	watchChan, err := k.Client.CoreV1().Namespaces().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), NamespaceWatcher{})

	return nil
}
