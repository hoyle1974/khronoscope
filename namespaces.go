package main

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type NamespaceWatchMe struct {
}

func (n NamespaceWatchMe) Tick() {
}

func (n NamespaceWatchMe) Kind() string {
	return "Namespace"
}

func (n NamespaceWatchMe) convert(obj runtime.Object) *corev1.Namespace {
	ret, ok := obj.(*corev1.Namespace)
	if !ok {
		return nil
	}
	return ret
}

func (n NamespaceWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n NamespaceWatchMe) Add(obj runtime.Object) Resource {
	namespace := n.convert(obj)
	return NewResource(namespace.ObjectMeta.CreationTimestamp.Time, n.Kind(), namespace.Namespace, namespace.Name, namespace)
}
func (n NamespaceWatchMe) Modified(obj runtime.Object) Resource {
	namespace := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), namespace.Namespace, namespace.Name, namespace)

}
func (n NamespaceWatchMe) Del(obj runtime.Object) Resource {
	namespace := n.convert(obj)
	return NewResource(namespace.ObjectMeta.DeletionTimestamp.Time, n.Kind(), namespace.Namespace, namespace.Name, namespace)
}

func watchForNamespaces(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching namespaces . . .")
	watchChan, err := k.client.CoreV1().Namespaces().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), NamespaceWatchMe{})
}
