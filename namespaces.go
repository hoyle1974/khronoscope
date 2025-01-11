package main

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type NamespacedRenderer struct {
}

func (r NamespacedRenderer) Render(resource Resource, details bool) []string {

	if details {
		// extra := resource.GetExtra()
		namespace := resource.Object.(*corev1.Namespace)

		// 		Name:         local-path-storage
		// Labels:       kubernetes.io/metadata.name=local-path-storage
		// Annotations:  <none>
		// Status:       Active
		out := []string{}
		out = append(out, "Name: "+resource.Name)
		out = append(out, fmt.Sprintf("Status: %v", namespace.Status))

		out = append(out, RenderMapOfStrings("Labels:", namespace.GetLabels())...)
		out = append(out, RenderMapOfStrings("Annotations:", namespace.GetAnnotations())...)

		return out

	}

	return []string{resource.Name}
}

type NamespaceWatchMe struct {
}

func (n NamespaceWatchMe) Tick() {
}

func (n NamespaceWatchMe) Kind() string {
	return "Namespace"
}

func (n *NamespaceWatchMe) Renderer() ResourceRenderer {
	return NamespacedRenderer{}
}

func (n NamespaceWatchMe) convert(obj runtime.Object) *corev1.Namespace {
	ret, ok := obj.(*corev1.Namespace)
	if !ok {
		return nil
	}
	return ret
}

func (n NamespaceWatchMe) Add(obj runtime.Object) Resource {
	namespace := n.convert(obj)
	return NewResource(string(namespace.ObjectMeta.GetUID()), namespace.ObjectMeta.CreationTimestamp.Time, n.Kind(), namespace.Namespace, namespace.Name, namespace, NamespacedRenderer{})
}

func (n NamespaceWatchMe) Modified(obj runtime.Object) Resource {
	namespace := n.convert(obj)
	return NewResource(string(namespace.ObjectMeta.GetUID()), time.Now(), n.Kind(), namespace.Namespace, namespace.Name, namespace, NamespacedRenderer{})
}

func (n NamespaceWatchMe) Del(obj runtime.Object) Resource {
	namespace := n.convert(obj)
	return NewResource(string(namespace.ObjectMeta.GetUID()), time.Now(), n.Kind(), namespace.Namespace, namespace.Name, namespace, NamespacedRenderer{})
}

func watchForNamespaces(watcher *K8sWatcher, k KhronosConn) {
	fmt.Println("Watching namespaces . . .")
	watchChan, err := k.client.CoreV1().Namespaces().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), NamespaceWatchMe{})
}
