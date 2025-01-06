package main

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DaemonSetWatchMe struct {
}

func (n DaemonSetWatchMe) Tick() {
}

func (n DaemonSetWatchMe) Kind() string {
	return "DeamonSet"
}

func (n DaemonSetWatchMe) convert(obj runtime.Object) *appsv1.DaemonSet {
	ret, ok := obj.(*appsv1.DaemonSet)
	if !ok {
		return nil
	}
	return ret
}

func (n DaemonSetWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n DaemonSetWatchMe) Add(obj runtime.Object) Resource {
	ds := n.convert(obj)
	return NewResource(ds.ObjectMeta.CreationTimestamp.Time, n.Kind(), ds.Namespace, ds.Name, ds)

}
func (n DaemonSetWatchMe) Modified(obj runtime.Object) Resource {
	ds := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), ds.Namespace, ds.Name, ds)

}
func (n DaemonSetWatchMe) Del(obj runtime.Object) Resource {
	ds := n.convert(obj)
	return NewResource(ds.ObjectMeta.DeletionTimestamp.Time, n.Kind(), ds.Namespace, ds.Name, ds)
}

func watchForDaemonSet(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching daemonset . . .")
	watchChan, err := k.client.AppsV1().DaemonSets("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), DaemonSetWatchMe{})
}
