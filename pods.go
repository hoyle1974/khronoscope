package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type PodRenderer struct {
	n *PodWatchMe
}

func (r PodRenderer) Render(resource Resource) string {
	extra := resource.GetExtra()

	out := ""
	e, ok := extra["Metrics"]
	if ok {
		out += " - "
		out += fmt.Sprintf("%v", e)
	}
	phase, ok := extra["Phase"]
	if ok {
		out += fmt.Sprintf(" [%v]", phase)
	}
	node, ok := extra["Node"]
	if ok {
		out += fmt.Sprintf(" Node:%s", node)
	}
	rt, ok := extra["StartTime"]
	if ok {
		out += fmt.Sprintf(" Up For:%s", time.Since(rt.(time.Time)).Truncate(time.Second))
	}
	return out
}

type PodWatchMe struct {
	k KhronosConn
	w *Watcher

	lastPodMetrics atomic.Pointer[v1beta1.PodMetricsList]
}

func calculatePercentage(usage int64, limit int64) float64 {
	if limit == 0 {
		return 0
	}
	return (float64(usage) / float64(limit)) * 100
}

func (n *PodWatchMe) getMetricsForPod(pod *corev1.Pod) map[string]string {
	metricsExtra := map[string]string{}
	lastPodMetrics := n.lastPodMetrics.Load()
	if lastPodMetrics == nil {
		return metricsExtra
	}
	for _, podMetrics := range lastPodMetrics.Items {
		if podMetrics.Namespace == pod.Namespace && podMetrics.Name == pod.Name {
			for _, container := range pod.Spec.Containers {
				for _, containerMetric := range podMetrics.Containers {
					if container.Name == containerMetric.Name {
						cpuUsage := containerMetric.Usage[corev1.ResourceCPU]
						memoryUsage := containerMetric.Usage[corev1.ResourceMemory]

						cpuLimit := container.Resources.Limits[corev1.ResourceCPU]
						memoryLimit := container.Resources.Limits[corev1.ResourceMemory]

						cpuPercentage := calculatePercentage(cpuUsage.MilliValue(), cpuLimit.MilliValue())
						memoryPercentage := calculatePercentage(memoryUsage.Value(), memoryLimit.Value())

						metricsExtra[container.Name] = fmt.Sprintf("CPU: %s | Memory: %s", renderProgressBar(cpuPercentage), renderProgressBar(memoryPercentage))
					}
				}
			}
			return metricsExtra
		}
	}
	return metricsExtra
}

func (n *PodWatchMe) updateResourceMetrics(resource Resource) {
	pod := resource.Object.(*corev1.Pod)

	metricsExtra := n.getMetricsForPod(pod)
	if len(metricsExtra) > 0 {
		resource = resource.SetExtraKV("Metrics", metricsExtra)
		resource.Timestamp = time.Now()
		n.w.Update(resource)
	}
}

func (n *PodWatchMe) Tick() {
	n.w.Log(fmt.Sprintf("Tick: %v", time.Now()))

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	m, err := n.k.mc.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	n.lastPodMetrics.Store(m)

	// // Get the current resources
	resources := n.w.GetStateAtTime(time.Now(), "Pod", "")
	for _, resource := range resources {
		n.updateResourceMetrics(resource)
	}
}

func (n *PodWatchMe) Kind() string {
	return "Pod"
}

func (n *PodWatchMe) Renderer() ResourceRenderer {
	return PodRenderer{n}
}

func (n *PodWatchMe) convert(obj runtime.Object) *corev1.Pod {
	ret, ok := obj.(*corev1.Pod)
	if !ok {
		return nil
	}
	return ret
}

func (n *PodWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n *PodWatchMe) getExtra(pod *corev1.Pod) map[string]any {
	extra := map[string]any{}
	extra["Phase"] = pod.Status.Phase
	extra["Node"] = pod.Spec.NodeName
	extra["Metrics"] = n.getMetricsForPod(pod)

	// Calculate the running time
	startTime := pod.Status.StartTime
	if startTime != nil {
		extra["StartTime"] = pod.Status.StartTime.Time
	}

	return extra
}

func (n *PodWatchMe) Add(obj runtime.Object) Resource {
	pod := n.convert(obj)
	return NewResource(pod.ObjectMeta.CreationTimestamp.Time, n.Kind(), pod.Namespace, pod.Name, pod, n.Renderer()).SetExtra(n.getExtra(pod))
}
func (n *PodWatchMe) Modified(obj runtime.Object) Resource {
	pod := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), pod.Namespace, pod.Name, pod, n.Renderer()).SetExtra(n.getExtra(pod))
}
func (n *PodWatchMe) Del(obj runtime.Object) Resource {
	pod := n.convert(obj)
	r := NewResource(time.Now() /*pod.DeletionTimestamp.Time*/, n.Kind(), pod.Namespace, pod.Name, pod, n.Renderer()).SetExtra(n.getExtra(pod))
	return r
}

func watchForPods(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching pods . . .")
	watchChan, err := k.client.CoreV1().Pods("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), &PodWatchMe{k: k, w: watcher})
}
