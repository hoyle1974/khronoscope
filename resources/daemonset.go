package resources

import (
	"context"
	"fmt"

	"github.com/hoyle1974/khronoscope/conn"
	"github.com/hoyle1974/khronoscope/internal/k8s"
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
	result = append(result,
		fmt.Sprintf("Name:           %s", ds.Name),
		fmt.Sprintf("Selector:       %s", k8s.FormatLabelSelector(ds.Spec.Selector)),
		fmt.Sprintf("Node-Selector:  %s", k8s.FormatNodeSelectors(ds.Spec.Template.Spec.NodeSelector)),
	)
	result = append(result, misc.RenderMapOfStrings("Labels:", ds.Labels)...)
	result = append(result, misc.RenderMapOfStrings("Annotations:", ds.Annotations)...)

	// Status information
	result = append(result,
		fmt.Sprintf("Desired Number of Nodes Scheduled: %d", ds.Status.DesiredNumberScheduled),
		fmt.Sprintf("Current Number of Nodes Scheduled: %d", ds.Status.CurrentNumberScheduled),
		fmt.Sprintf("Number of Nodes Scheduled with Up-to-date Pods: %d", ds.Status.UpdatedNumberScheduled),
		fmt.Sprintf("Number of Nodes Scheduled with Available Pods: %d", ds.Status.NumberAvailable),
		fmt.Sprintf("Number of Nodes Misscheduled: %d", ds.Status.NumberMisscheduled),
	)

	// Pod status
	result = append(result, fmt.Sprintf("Pods Status:    %d Running / %d Ready / %d Updated / %d Available",
		ds.Status.CurrentNumberScheduled,
		ds.Status.NumberReady,
		ds.Status.UpdatedNumberScheduled,
		ds.Status.NumberAvailable,
	))

	// Pod template details
	result = append(result, "Pod Template:")
	result = append(result, misc.RenderMapOfStrings("  Labels:", ds.Spec.Template.Labels)...)
	if ds.Spec.Template.Spec.ServiceAccountName != "" {
		result = append(result, fmt.Sprintf("  Service Account:  %s", ds.Spec.Template.Spec.ServiceAccountName))
	}

	// Containers info
	if len(ds.Spec.Template.Spec.Containers) > 0 {
		result = append(result, "  Containers:")
		for _, container := range ds.Spec.Template.Spec.Containers {
			result = append(result,
				fmt.Sprintf("    %s:", container.Name),
				fmt.Sprintf("      Image:       %s", container.Image),
				fmt.Sprintf("      Ports:       %s", k8s.FormatPorts(container.Ports)),
				fmt.Sprintf("      Limits:      %s", k8s.FormatLimits(container.Resources.Limits)),
				fmt.Sprintf("      Requests:    %s", k8s.FormatLimits(container.Resources.Requests)),
				fmt.Sprintf("      Environment: %s", k8s.FormatEnvironment(container.Env)),
				fmt.Sprintf("      Mounts:      %s", k8s.FormatVolumeMounts(container.VolumeMounts)),
			)

			if container.LivenessProbe != nil {
				result = append(result, fmt.Sprintf("      Liveness:    %s", k8s.FormatLiveness(container.LivenessProbe)))
			}
			if container.ReadinessProbe != nil {
				result = append(result, fmt.Sprintf("      Readiness:   %s", k8s.FormatLiveness(container.ReadinessProbe)))
			}
		}
	}

	// Volumes
	if len(ds.Spec.Template.Spec.Volumes) > 0 {
		result = append(result, "  Volumes:")
		for _, volume := range ds.Spec.Template.Spec.Volumes {
			result = append(result, fmt.Sprintf("    %s: %s", volume.Name, k8s.FormatVolumeSource(volume.VolumeSource)))
		}
	}

	// Node Selectors and Tolerations
	result = append(result,
		fmt.Sprintf("  Node-Selectors: %s", k8s.FormatNodeSelectors(ds.Spec.Template.Spec.NodeSelector)),
		fmt.Sprintf("  Tolerations:    %s", k8s.FormatTolerations(ds.Spec.Template.Spec.Tolerations)),
	)

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

func (n DaemonSetWatcher) Renderer() ResourceRenderer {
	return DaemonSetRenderer{}
}

func (n DaemonSetWatcher) convert(obj runtime.Object) *appsv1.DaemonSet {
	if obj == nil {
		return nil
	}
	if ds, ok := obj.(*appsv1.DaemonSet); ok {
		return ds
	}
	return nil
}

func (n DaemonSetWatcher) ToResource(obj runtime.Object) Resource {
	ds := n.convert(obj)
	if ds == nil {
		return Resource{}
	}
	return Resource{
		Uid:     string(ds.UID),
		Name:    ds.Name,
		Details: formatDaemonSetDetails(ds),
	}
}

func (n DaemonSetWatcher) Kind() string {
	return "DaemonSet"
}

func watchForDaemonSet(watcher *K8sWatcher, k conn.KhronosConn) {
	w, err := k.Client.AppsV1().DaemonSets("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.registerEventWatcher(w.ResultChan(), DaemonSetWatcher{})

}
