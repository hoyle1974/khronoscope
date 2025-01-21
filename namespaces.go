package main

import (
	"context"
	"fmt"

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

type NamespaceWatcher struct {
}

func (n NamespaceWatcher) Tick() {
}

func (n NamespaceWatcher) Kind() string {
	return "Namespace"
}

func (n NamespaceWatcher) Renderer() ResourceRenderer {
	return NamespacedRenderer{}
}

func (n NamespaceWatcher) convert(obj runtime.Object) *corev1.Namespace {
	ret, ok := obj.(*corev1.Namespace)
	if !ok {
		return nil
	}
	return ret
}

func (n NamespaceWatcher) ToResource(obj runtime.Object) Resource {
	return NewK8sResource(n.Kind(), n.convert(obj))
}

func watchForNamespaces(watcher *K8sWatcher, k KhronosConn) {
	watchChan, err := k.client.CoreV1().Namespaces().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), NamespaceWatcher{})
}
