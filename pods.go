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

type PodWatchMe struct {
	k KhronosConn
}

func (n PodWatchMe) Kind() string {
	return "Pod"
}

func (n PodWatchMe) convert(obj runtime.Object) *corev1.Pod {
	ret, ok := obj.(*corev1.Pod)
	if !ok {
		return nil
	}
	return ret
}

func (n PodWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n PodWatchMe) getExtra(pod *corev1.Pod) map[string]any {
	extra := map[string]any{}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	m, err := n.k.mc.MetricsV1beta1().PodMetricses(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err == nil {
		extra["PodMetrics"] = m
	}

	extra["Phase"] = pod.Status.Phase
	return extra
}

func (n PodWatchMe) update(obj runtime.Object) *Resource {
	pod := n.convert(obj)

	o, err := n.k.client.CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, v1.GetOptions{})
	if err != nil {
		return nil
	}

	r := n.Modified(o)
	return &r
}

func (n PodWatchMe) Add(obj runtime.Object) Resource {
	pod := n.convert(obj)
	return NewResource(pod.ObjectMeta.CreationTimestamp.Time, n.Kind(), pod.Namespace, pod.Name, pod).SetExtra(n.getExtra(pod)).SetUpdate(func() *Resource { return n.update(obj) })
}
func (n PodWatchMe) Modified(obj runtime.Object) Resource {
	pod := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), pod.Namespace, pod.Name, pod).SetExtra(n.getExtra(pod)).SetUpdate(func() *Resource { return n.update(obj) })
}
func (n PodWatchMe) Del(obj runtime.Object) Resource {
	pod := n.convert(obj)
	r := NewResource(time.Now() /*pod.DeletionTimestamp.Time*/, n.Kind(), pod.Namespace, pod.Name, pod).SetExtra(n.getExtra(pod))
	return r
}

func watchForPods(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching pods . . .")
	watchChan, err := k.client.CoreV1().Pods("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), PodWatchMe{k})
}
