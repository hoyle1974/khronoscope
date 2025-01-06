package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ReplicaSetWatchMe struct {
}

func (n ReplicaSetWatchMe) Kind() string {
	return "ReplicaSet"
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

func (n ReplicaSetWatchMe) Add(obj runtime.Object) Resource {
	rs := n.convert(obj)
	return NewResource(rs.ObjectMeta.CreationTimestamp.Time, n.Kind(), rs.Namespace, rs.Name, rs)

}
func (n ReplicaSetWatchMe) Modified(obj runtime.Object) Resource {
	rs := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), rs.Namespace, rs.Name, rs)

}
func (n ReplicaSetWatchMe) Del(obj runtime.Object) Resource {
	rs := n.convert(obj)
	return NewResource(rs.ObjectMeta.DeletionTimestamp.Time, n.Kind(), rs.Namespace, rs.Name, rs)

}

func watchForReplicaSet(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching replica set . . .")
	watchChan, err := k.client.AppsV1().ReplicaSets("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), ReplicaSetWatchMe{})
}
