package resources

import (
	"context"

	"github.com/hoyle1974/khronoscope/conn"
	"github.com/hoyle1974/khronoscope/internal/ui"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type NamespacedRenderer struct {
}

func (r NamespacedRenderer) Render(resource Resource, details bool) []string {

	if details {
		return resource.Details
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

func (n NamespaceWatcher) ToResource(obj runtime.Object) Resource {
	return NewK8sResource(n.Kind(), n.convert(obj), ui.FormatNamespaceDetails(n.convert(obj)), nil)
}

func watchForNamespaces(watcher *K8sWatcher, k conn.KhronosConn) {
	watchChan, err := k.Client.CoreV1().Namespaces().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), NamespaceWatcher{})
}
