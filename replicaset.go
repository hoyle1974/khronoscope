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

func (r ReplicaSetRenderer) Render(resource Resource, details bool) []string {
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
	return NewResource(rs.ObjectMeta.CreationTimestamp.Time, n.Kind(), rs.Namespace, rs.Name, rs, n.Renderer()).SetExtra(n.getExtra(rs))

}
func (n ReplicaSetWatchMe) Modified(obj runtime.Object) Resource {
	rs := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), rs.Namespace, rs.Name, rs, n.Renderer()).SetExtra(n.getExtra(rs))

}
func (n ReplicaSetWatchMe) Del(obj runtime.Object) Resource {
	rs := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), rs.Namespace, rs.Name, rs, n.Renderer()).SetExtra(n.getExtra(rs))

}

func watchForReplicaSet(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching replica set . . .")
	watchChan, err := k.client.AppsV1().ReplicaSets("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), ReplicaSetWatchMe{})
}
