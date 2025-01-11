package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DeploymentRenderer struct {
}

func formatDeploymentDetails(deployment *appsv1.Deployment) []string {
	var result []string

	// Basic details
	result = append(result, fmt.Sprintf("Name:           %s", deployment.Name))
	result = append(result, fmt.Sprintf("Namespace:      %s", deployment.Namespace))
	result = append(result, fmt.Sprintf("Selector:       %s", deployment.Spec.Selector))
	result = append(result, RenderMapOfStrings("Labels:", deployment.Labels)...)
	result = append(result, RenderMapOfStrings("Annotations:", deployment.Annotations)...)

	// Replicas
	result = append(result, fmt.Sprintf("Replicas:       %d current / %d desired", deployment.Status.Replicas, *deployment.Spec.Replicas))

	// Pods Status
	podStatus := fmt.Sprintf("%d Running / 0 Waiting / 0 Succeeded / 0 Failed", deployment.Status.Replicas) // Placeholder, needs actual pod status logic
	result = append(result, fmt.Sprintf("Pods Status:    %s", podStatus))

	// Pod template details
	result = append(result, "Pod Template:")
	result = append(result, RenderMapOfStrings("  Labels:", deployment.Spec.Template.Labels)...)

	// Containers info
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		result = append(result, "  Containers:")
		for _, container := range deployment.Spec.Template.Spec.Containers {
			result = append(result, fmt.Sprintf("    %s:", container.Name))
			result = append(result, fmt.Sprintf("      Image:       %s", container.Image))
			result = append(result, fmt.Sprintf("      Port:        %v", container.Ports))
			// result = append(result, fmt.Sprintf("      Host Port:   %v", container.HostPorts))
			result = append(result, fmt.Sprintf("      Limits:      %s", formatLimits(container.Resources.Limits)))
			result = append(result, fmt.Sprintf("      Requests:    %s", formatLimits(container.Resources.Requests)))
			result = append(result, fmt.Sprintf("      Environment: %s", formatEnvironment(container.Env)))
			result = append(result, fmt.Sprintf("      Mounts:      %s", formatVolumeMounts(container.VolumeMounts)))
		}
	}

	// Volumes
	if len(deployment.Spec.Template.Spec.Volumes) > 0 {
		result = append(result, "  Volumes:")
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			result = append(result, fmt.Sprintf("    %s: %s", volume.Name, formatVolumeSource(volume.VolumeSource)))
		}
	}

	// Node Selectors
	result = append(result, fmt.Sprintf("  Node-Selectors:       %s", formatNodeSelectors(deployment.Spec.Template.Spec.NodeSelector)))

	// Tolerations
	result = append(result, fmt.Sprintf("  Tolerations:          %s", formatTolerations(deployment.Spec.Template.Spec.Tolerations)))

	// Events
	result = append(result, "Events:                 <none>")

	return result
}

func (r DeploymentRenderer) Render(resource Resource, details bool) []string {
	if details {
		return formatDeploymentDetails(resource.Object.(*appsv1.Deployment))
	}

	return []string{resource.Key()}
}

type DeploymentWatchMe struct {
}

func (n DeploymentWatchMe) Tick() {
}

func (n DeploymentWatchMe) Kind() string {
	return "Deployment"
}

func (n *DeploymentWatchMe) Renderer() ResourceRenderer {
	return nil
}

func (n DeploymentWatchMe) convert(obj runtime.Object) *appsv1.Deployment {
	ret, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil
	}
	return ret
}

func (n DeploymentWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n DeploymentWatchMe) Add(obj runtime.Object) Resource {
	d := n.convert(obj)
	return NewResource(string(d.ObjectMeta.GetUID()), d.ObjectMeta.CreationTimestamp.Time, n.Kind(), d.Namespace, d.Name, d, DeploymentRenderer{})
}
func (n DeploymentWatchMe) Modified(obj runtime.Object) Resource {
	d := n.convert(obj)
	return NewResource(string(d.ObjectMeta.GetUID()), time.Now(), n.Kind(), d.Namespace, d.Name, d, DeploymentRenderer{})

}
func (n DeploymentWatchMe) Del(obj runtime.Object) Resource {
	d := n.convert(obj)
	return NewResource(string(d.ObjectMeta.GetUID()), time.Now(), n.Kind(), d.Namespace, d.Name, d, DeploymentRenderer{})
}

func watchForDeployments(watcher *K8sWatcher, k KhronosConn) {
	watchChan, err := k.client.AppsV1().Deployments("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), DeploymentWatchMe{})
}
