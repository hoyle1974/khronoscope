package resources

import (
	"context"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/ui"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DeploymentRenderer struct {
}

func (r DeploymentRenderer) Render(resource Resource, details bool) []string {
	if details {
		return resource.Details
	}

	return []string{resource.Key()}
}

type DeploymentWatcher struct {
}

func (n DeploymentWatcher) Tick() {
}

func (n DeploymentWatcher) Kind() string {
	return "Deployment"
}

func (n DeploymentWatcher) Renderer() ResourceRenderer {
	return DeploymentRenderer{}
}

func (n DeploymentWatcher) convert(obj runtime.Object) *appsv1.Deployment {
	ret, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil
	}
	return ret
}

func (n DeploymentWatcher) ToResource(obj runtime.Object) Resource {
	return NewK8sResource(n.Kind(), n.convert(obj), ui.FormatDeploymentDetails(n.convert(obj)), nil)
}

func watchForDeployments(watcher *K8sWatcher, k conn.KhronosConn) error {
	watchChan, err := k.Client.AppsV1().Deployments("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), DeploymentWatcher{})

	return nil

}
