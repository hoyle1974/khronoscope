package resources

import (
	"context"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc/format"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ServiceRenderer struct {
}

func (r ServiceRenderer) Render(resource Resource, obj any, details bool) []string {
	if details {
		return format.FormatServiceDetails(obj.(*corev1.Service))
	}

	return []string{resource.Key()}
}

type ServiceWatcher struct {
}

func (n ServiceWatcher) Tick() {
}

func (n ServiceWatcher) Kind() string {
	return "Service"
}

func (n ServiceWatcher) Renderer() ResourceRenderer {
	return nil
}

func (n ServiceWatcher) convert(obj runtime.Object) *corev1.Service {
	ret, ok := obj.(*corev1.Service)
	if !ok {
		return nil
	}
	return ret
}

func (n ServiceWatcher) ToResource(obj runtime.Object) Resource {
	return NewK8sResource(n.Kind(), n.convert(obj), format.FormatServiceDetails(n.convert(obj)), nil)
}

func watchForService(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	watchChan, err := k.Client.CoreV1().Services(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), ServiceWatcher{})

	return nil
}
