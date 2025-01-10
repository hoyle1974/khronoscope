package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ReplicaSetRenderer struct {
	n *ReplicaSetWatchMe
}

func formatReplicaSetDetails(rs *appsv1.ReplicaSet) []string {
	var result []string

	// Basic details
	result = append(result, fmt.Sprintf("Name:           %s", rs.Name))
	result = append(result, fmt.Sprintf("Namespace:      %s", rs.Namespace))
	result = append(result, fmt.Sprintf("Selector:       %s", rs.Spec.Selector))

	result = append(result, RenderMapOfStrings("Labels:", rs.Labels)...)
	result = append(result, RenderMapOfStrings("Annotations:", rs.Annotations)...)

	// Controlled By
	if rs.OwnerReferences != nil && len(rs.OwnerReferences) > 0 {
		result = append(result, fmt.Sprintf("Controlled By:  %s", rs.OwnerReferences[0].Kind+"/"+rs.OwnerReferences[0].Name))
	}

	// Replicas
	result = append(result, fmt.Sprintf("Replicas:       %d current / %d desired", rs.Status.Replicas, rs.Spec.Replicas))

	// Pod status
	podStatus := "0 Running / 0 Waiting / 0 Succeeded / 0 Failed" // Placeholder, needs actual pod status logic
	result = append(result, fmt.Sprintf("Pods Status:    %s", podStatus))

	// Pod template details
	result = append(result, "Pod Template:")
	result = append(result, RenderMapOfStrings("  Labels:", rs.Spec.Template.Labels)...)
	result = append(result, RenderMapOfStrings("  Annotations:", rs.Spec.Template.Annotations)...)
	// result = append(result, fmt.Sprintf("  Service Account:  %s", rs.Spec.Template.ServiceAccountName)) // Corrected here

	// Containers info
	if len(rs.Spec.Template.Spec.Containers) > 0 {
		result = append(result, "  Containers:")
		for _, container := range rs.Spec.Template.Spec.Containers {
			result = append(result, fmt.Sprintf("    %s:", container.Name))
			result = append(result, fmt.Sprintf("      Image:       %s", container.Image))
			result = append(result, fmt.Sprintf("      Ports:       %s", formatPorts(container.Ports)))
			result = append(result, fmt.Sprintf("      Args:        %s", formatArgs(container.Args)))
			result = append(result, fmt.Sprintf("      Limits:      %s", formatLimits(container.Resources.Limits)))
			result = append(result, fmt.Sprintf("      Requests:    %s", formatLimits(container.Resources.Requests)))
			result = append(result, fmt.Sprintf("      Liveness:    %s", formatLiveness(container.LivenessProbe)))
			result = append(result, fmt.Sprintf("      Readiness:   %s", formatLiveness(container.ReadinessProbe)))
			result = append(result, fmt.Sprintf("      Environment: %s", formatEnvironment(container.Env)))
			result = append(result, fmt.Sprintf("      Mounts:      %s", formatVolumeMounts(container.VolumeMounts)))
		}
	}

	// Volumes
	if len(rs.Spec.Template.Spec.Volumes) > 0 {
		result = append(result, "  Volumes:")
		for _, volume := range rs.Spec.Template.Spec.Volumes {
			result = append(result, fmt.Sprintf("    %s: %s", volume.Name, "")) //volume.VolumeSource.ConfigMap.Name))
		}
	}

	// Priority Class Name
	result = append(result, fmt.Sprintf("  Priority Class Name:  %s", rs.Spec.Template.Spec.PriorityClassName))

	// Node Selectors
	result = append(result, fmt.Sprintf("  Node-Selectors:       %s", formatNodeSelectors(rs.Spec.Template.Spec.NodeSelector)))

	// Tolerations
	result = append(result, fmt.Sprintf("  Tolerations:          %s", formatTolerations(rs.Spec.Template.Spec.Tolerations)))

	// Events
	result = append(result, "Events:                 <none>")

	return result
}

func (r ReplicaSetRenderer) Render(resource Resource, details bool) []string {

	if details {
		rs := resource.Object.(*appsv1.ReplicaSet)

		return formatReplicaSetDetails(rs)
	}

	extra := ""
	e, ok := resource.GetExtra()["Status"]
	if ok {
		s := e.(appsv1.ReplicaSetStatus)
		extra += fmt.Sprintf(" - Replicas:%d Available:%d Ready:%d FullyLabeledReplicas:%d", s.Replicas, s.AvailableReplicas, s.ReadyReplicas, s.FullyLabeledReplicas)
	}
	return []string{extra}
}

type ReplicaSetWatchMe struct {
}

func (n ReplicaSetWatchMe) Tick() {
}

func (n ReplicaSetWatchMe) Kind() string {
	return "ReplicaSet"
}

func (n *ReplicaSetWatchMe) Renderer() ResourceRenderer {
	return ReplicaSetRenderer{n}
}

func (n ReplicaSetWatchMe) convert(obj runtime.Object) *appsv1.ReplicaSet {
	ret, ok := obj.(*appsv1.ReplicaSet)
	if !ok {
		return nil
	}
	return ret
}

func (n ReplicaSetWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n ReplicaSetWatchMe) getExtra(rs *appsv1.ReplicaSet) map[string]any {
	extra := map[string]any{}

	extra["Status"] = rs.Status

	return extra
}

func (n ReplicaSetWatchMe) Add(obj runtime.Object) Resource {
	rs := n.convert(obj)
	return NewResource(string(rs.ObjectMeta.GetUID()), rs.ObjectMeta.CreationTimestamp.Time, n.Kind(), rs.Namespace, rs.Name, rs, n.Renderer()).SetExtra(n.getExtra(rs))

}
func (n ReplicaSetWatchMe) Modified(obj runtime.Object) Resource {
	rs := n.convert(obj)
	return NewResource(string(rs.ObjectMeta.GetUID()), time.Now(), n.Kind(), rs.Namespace, rs.Name, rs, n.Renderer()).SetExtra(n.getExtra(rs))

}
func (n ReplicaSetWatchMe) Del(obj runtime.Object) Resource {
	rs := n.convert(obj)
	return NewResource(string(rs.ObjectMeta.GetUID()), time.Now(), n.Kind(), rs.Namespace, rs.Name, rs, n.Renderer()).SetExtra(n.getExtra(rs))

}

func watchForReplicaSet(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching replica set . . .")
	watchChan, err := k.client.AppsV1().ReplicaSets("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), ReplicaSetWatchMe{})
}
