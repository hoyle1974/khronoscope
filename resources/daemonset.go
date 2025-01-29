package resources

import (
	"context"

	"github.com/hoyle1974/khronoscope/conn"
	"github.com/hoyle1974/khronoscope/internal/ui"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DaemonSetRenderer struct {
}

func (r DaemonSetRenderer) Render(resource Resource, details bool) []string {
	if details {
		return resource.Details
	}

	return []string{resource.Key()}
}

type DaemonSetWatcher struct {
}

func (n DaemonSetWatcher) Tick() {
}

func (n DaemonSetWatcher) Kind() string {
	return "DaemonSet"
}

func (n DaemonSetWatcher) Renderer() ResourceRenderer {
	return DaemonSetRenderer{}
}

func (n DaemonSetWatcher) convert(obj runtime.Object) *appsv1.DaemonSet {
	ret, ok := obj.(*appsv1.DaemonSet)
	if !ok {
		return nil
	}
	return ret
}

func (n DaemonSetWatcher) ToResource(obj runtime.Object) Resource {
	return NewK8sResource(n.Kind(), n.convert(obj), ui.FormatDaemonSetDetails(n.convert(obj)), nil)
}

func watchForDaemonSet(watcher *K8sWatcher, k conn.KhronosConn) error {
	watchChan, err := k.Client.AppsV1().DaemonSets("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), DaemonSetWatcher{})

	return nil
}
