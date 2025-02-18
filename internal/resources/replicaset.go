package resources

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/misc/format"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ReplicaSetExtra struct {
	Name                 string
	Namespace            string
	Labels               []string
	Annotations          []string
	Replicas             int32
	AvailableReplicas    int32
	ReadyReplicas        int32
	FullyLabeledReplicas int32
	ObservedGeneration   int64
	CreationTimestamp    string
	OwnerReferences      []string
}

func (p ReplicaSetExtra) Copy() Copyable {
	return ReplicaSetExtra{
		Name:                 p.Name,
		Namespace:            p.Namespace,
		Labels:               misc.DeepCopyArray(p.Labels),
		Annotations:          misc.DeepCopyArray(p.Annotations),
		Replicas:             p.Replicas,
		AvailableReplicas:    p.AvailableReplicas,
		ReadyReplicas:        p.ReadyReplicas,
		FullyLabeledReplicas: p.FullyLabeledReplicas,
		ObservedGeneration:   p.ObservedGeneration,
		CreationTimestamp:    p.CreationTimestamp,
		OwnerReferences:      misc.DeepCopyArray(p.OwnerReferences),
	}
}

func newReplicaSetExtra(rs *appsv1.ReplicaSet) ReplicaSetExtra {
	var ownerRefs []string
	for _, owner := range rs.OwnerReferences {
		ownerRefs = append(ownerRefs, fmt.Sprintf("%s/%s", owner.Kind, owner.Name))
	}

	return ReplicaSetExtra{
		Name:                 rs.Name,
		Namespace:            rs.Namespace,
		Labels:               misc.RenderMapOfStrings("Labels", rs.Labels),
		Annotations:          misc.RenderMapOfStrings("Annotations", rs.Annotations),
		Replicas:             *rs.Spec.Replicas,
		AvailableReplicas:    rs.Status.AvailableReplicas,
		ReadyReplicas:        rs.Status.ReadyReplicas,
		FullyLabeledReplicas: rs.Status.FullyLabeledReplicas,
		ObservedGeneration:   rs.Status.ObservedGeneration,
		CreationTimestamp:    rs.CreationTimestamp.Format("2006-01-02T15:04:05Z"),
		OwnerReferences:      ownerRefs,
	}
}

func renderReplicaSetExtra(extra ReplicaSetExtra) []string {
	var output []string
	output = append(output, fmt.Sprintf("NAME: %s", extra.Name))
	output = append(output, fmt.Sprintf("NAMESPACE: %s", extra.Namespace))
	output = append(output, fmt.Sprintf("REPLICAS: %d", extra.Replicas))
	output = append(output, fmt.Sprintf("AVAILABLE: %d", extra.AvailableReplicas))
	output = append(output, fmt.Sprintf("READY: %d", extra.ReadyReplicas))
	output = append(output, fmt.Sprintf("FULLY LABELED: %d", extra.FullyLabeledReplicas))
	output = append(output, fmt.Sprintf("OBSERVED GENERATION: %d", extra.ObservedGeneration))
	output = append(output, fmt.Sprintf("CREATED AT: %s", extra.CreationTimestamp))

	if len(extra.OwnerReferences) > 0 {
		sort.Strings(extra.OwnerReferences)
		output = append(output, fmt.Sprintf("OWNER REFERENCES: %s", strings.Join(extra.OwnerReferences, ", ")))
	}

	output = append(output, extra.Labels...)
	output = append(output, extra.Annotations...)

	return output
}

type ReplicaSetRenderer struct {
	// n *ReplicaSetWatcher
}

func (r ReplicaSetRenderer) Render(resource Resource, details bool) []string {
	extra := resource.Extra.(ReplicaSetExtra)

	if details {
		return renderReplicaSetExtra(extra)
	}

	return []string{fmt.Sprintf("%s - %d/%d/%d/%d", resource.Name, extra.Replicas, extra.AvailableReplicas, extra.ReadyReplicas, extra.FullyLabeledReplicas)}

}

type ReplicaSetWatcher struct {
}

func (n ReplicaSetWatcher) Tick() {
}

func (n ReplicaSetWatcher) Kind() string {
	return "ReplicaSet"
}

func (n ReplicaSetWatcher) Renderer() ResourceRenderer {
	return ReplicaSetRenderer{}
}

func (n ReplicaSetWatcher) convert(obj runtime.Object) *appsv1.ReplicaSet {
	ret, ok := obj.(*appsv1.ReplicaSet)
	if !ok {
		return nil
	}
	return ret
}

func (n ReplicaSetWatcher) ToResource(obj runtime.Object) Resource {
	rs := n.convert(obj)
	extra := newReplicaSetExtra(rs)
	return NewK8sResource(n.Kind(), rs, format.FormatReplicaSetDetails(rs), extra)
}

func watchForReplicaSet(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	watchChan, err := k.Client.AppsV1().ReplicaSets(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), ReplicaSetWatcher{})

	return nil
}
