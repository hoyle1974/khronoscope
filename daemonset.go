package main

import (
	"context"
	"fmt"

	"github.com/hoyle1974/khronoscope/internal/format"
	"github.com/hoyle1974/khronoscope/internal/misc"
	appsv1 "k8s.io/api/apps/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DaemonSetRenderer struct {
}

func formatDaemonSetDetails(ds *appsv1.DaemonSet) []string {
	var result []string
	if ds == nil {
		return result
	}

	// Basic details
	result = append(result, fmt.Sprintf("Name:           %s", ds.Name))
	result = append(result, fmt.Sprintf("Selector:       %s", ds.Spec.Selector))
	result = append(result, fmt.Sprintf("Node-Selector:  %s", format.NodeSelectors(ds.Spec.Template.Spec.NodeSelector)))
	result = append(result, misc.RenderMapOfStrings("Labels:", ds.Labels)...)
	result = append(result, misc.RenderMapOfStrings("Annotations:", ds.Annotations)...)

	// Desired Number of Nodes Scheduled
	result = append(result, fmt.Sprintf("Desired Number of Nodes Scheduled: %d", ds.Status.DesiredNumberScheduled))

	// Current Number of Nodes Scheduled
	result = append(result, fmt.Sprintf("Current Number of Nodes Scheduled: %d", ds.Status.CurrentNumberScheduled))

	// Nodes with up-to-date Pods
	result = append(result, fmt.Sprintf("Number of Nodes Scheduled with Up-to-date Pods: %d", ds.Status.NumberAvailable))

	// Nodes with available Pods
	result = append(result, fmt.Sprintf("Number of Nodes Scheduled with Available Pods: %d", ds.Status.NumberAvailable))

	// Nodes Misscheduled
	result = append(result, fmt.Sprintf("Number of Nodes Misscheduled: %d", ds.Status.NumberMisscheduled))

	// Pod status
	podStatus := fmt.Sprintf("%d Running / 0 Waiting / 0 Succeeded / 0 Failed", ds.Status.DesiredNumberScheduled) // Placeholder, needs actual pod status logic
	result = append(result, fmt.Sprintf("Pods Status:    %s", podStatus))

	// Pod template details
	result = append(result, "Pod Template:")
	result = append(result, misc.RenderMapOfStrings("  Labels:", ds.Spec.Template.Labels)...)
	// result = append(result, fmt.Sprintf("  Service Account:  %s", ds.Spec.Template.ServiceAccountName))

	// Containers info
	if len(ds.Spec.Template.Spec.Containers) > 0 {
		result = append(result, "  Containers:")
		for _, container := range ds.Spec.Template.Spec.Containers {
			result = append(result, fmt.Sprintf("    %s:", container.Name))
			result = append(result, fmt.Sprintf("      Image:       %s", container.Image))
			result = append(result, fmt.Sprintf("      Port:        %v", container.Ports))
			// result = append(result, fmt.Sprintf("      Host Port:   %v", container.HostPorts))
			result = append(result, fmt.Sprintf("      Limits:      %s", format.Limits(container.Resources.Limits)))
			result = append(result, fmt.Sprintf("      Requests:    %s", format.Limits(container.Resources.Requests)))
			result = append(result, fmt.Sprintf("      Environment: %s", format.Environment(container.Env)))
			result = append(result, fmt.Sprintf("      Mounts:      %s", format.VolumeMounts(container.VolumeMounts)))
		}
	}

	// Volumes
	if len(ds.Spec.Template.Spec.Volumes) > 0 {
		result = append(result, "  Volumes:")
		for _, volume := range ds.Spec.Template.Spec.Volumes {
			result = append(result, fmt.Sprintf("    %s: %s", volume.Name, format.VolumeSource(volume.VolumeSource)))
		}
	}

	// Node Selectors
	result = append(result, fmt.Sprintf("  Node-Selectors:       %s", format.NodeSelectors(ds.Spec.Template.Spec.NodeSelector)))

	// Tolerations
	result = append(result, fmt.Sprintf("  Tolerations:          %s", format.Tolerations(ds.Spec.Template.Spec.Tolerations)))

	return result
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
	return "DeamonSet"
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
	return NewK8sResource(n.Kind(), n.convert(obj), formatDaemonSetDetails(n.convert(obj)), nil)
}

func watchForDaemonSet(watcher *K8sWatcher, k KhronosConn) {
	watchChan, err := k.client.AppsV1().DaemonSets("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), DaemonSetWatcher{})
}
