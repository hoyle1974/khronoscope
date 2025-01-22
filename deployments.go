package main

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DeploymentRenderer struct {
}

func formatDeploymentDetails(deployment *appsv1.Deployment) []string {
	var result []string
	if deployment == nil {
		return result
	}

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
	return NewK8sResource(n.Kind(), n.convert(obj), formatDeploymentDetails(n.convert(obj)), nil)
}

func watchForDeployments(watcher *K8sWatcher, k KhronosConn) {
	watchChan, err := k.client.AppsV1().Deployments("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), DeploymentWatcher{})
}
