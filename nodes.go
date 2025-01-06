package main

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type NodeWatchMe struct {
	k KhronosConn
}

func (n NodeWatchMe) Kind() string {
	return "Node"
}

func (n NodeWatchMe) convert(obj runtime.Object) *corev1.Node {
	ret, ok := obj.(*corev1.Node)
	if !ok {
		return nil
	}
	return ret
}

func (n NodeWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n NodeWatchMe) getNodeExtra(node *corev1.Node) map[string]any {
	extra := map[string]any{}

	m, err := n.k.mc.MetricsV1beta1().NodeMetricses().Get(context.Background(), node.Name, metav1.GetOptions{})
	if err != nil {
		return extra
	}

	extra["NodeMetrics"] = m

	return extra
}

func (n NodeWatchMe) update(obj runtime.Object) Resource {
	return n.Modified(obj)
}

func (n NodeWatchMe) Add(obj runtime.Object) Resource {
	node := n.convert(obj)
	return NewResource(node.ObjectMeta.CreationTimestamp.Time, n.Kind(), node.Namespace, node.Name, node).SetExtra(n.getNodeExtra(node)).SetUpdate(func() Resource { return n.update(obj) })
}
func (n NodeWatchMe) Modified(obj runtime.Object) Resource {
	node := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), node.Namespace, node.Name, node).SetExtra(n.getNodeExtra(node)).SetUpdate(func() Resource { return n.update(obj) })
}
func (n NodeWatchMe) Del(obj runtime.Object) Resource {
	node := n.convert(obj)
	return NewResource(node.ObjectMeta.DeletionTimestamp.Time, n.Kind(), node.Namespace, node.Name, node).SetExtra(n.getNodeExtra(node))

}

func watchForNodes(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching nodes . . .")
	watchChan, err := k.client.CoreV1().Nodes().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), NodeWatchMe{k})
}
