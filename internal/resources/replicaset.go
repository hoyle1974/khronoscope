package resources

import (
	"context"
	"fmt"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/ui"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ReplicaSetExtra struct {
	Replicas             int32
	AvailableReplicas    int32
	ReadyReplicas        int32
	FullyLabeledReplicas int32
}

type ReplicaSetRenderer struct {
	// n *ReplicaSetWatcher
}

func (r ReplicaSetRenderer) Render(resource Resource, details bool) []string {

	if details {
		return resource.Details
	}

	extra := resource.Extra.(ReplicaSetExtra)
	return []string{fmt.Sprintf("%s - Replicas:%d Available:%d Ready:%d FullyLabeledReplicas:%d", resource.Name, extra.Replicas, extra.AvailableReplicas, extra.ReadyReplicas, extra.FullyLabeledReplicas)}

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

	extra := ReplicaSetExtra{
		Replicas:             rs.Status.Replicas,
		AvailableReplicas:    rs.Status.AvailableReplicas,
		ReadyReplicas:        rs.Status.ReadyReplicas,
		FullyLabeledReplicas: rs.Status.FullyLabeledReplicas,
	}

	r := NewK8sResource(n.Kind(), rs, ui.FormatReplicaSetDetails(rs), extra)

	return r
}

func watchForReplicaSet(watcher *K8sWatcher, k conn.KhronosConn) error {
	watchChan, err := k.Client.AppsV1().ReplicaSets("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), ReplicaSetWatcher{})

	return nil
}
