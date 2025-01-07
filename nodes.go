package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type NodeRenderer struct {
	n *NodeWatchMe
}

func (r NodeRenderer) Render(resource Resource) []string {
	out := ""
	extra := resource.GetExtra()
	e, ok := extra["Metrics"]
	if ok {
		m := e.(map[string]string)
		out += " - "
		out += fmt.Sprintf("%v", m[resource.Name])
	}
	rt, ok := extra["StartTime"]
	if ok {
		out += fmt.Sprintf(" Uptime:%s", time.Since(rt.(time.Time)).Truncate(time.Second))
	}
	return []string{out}
}

type NodeWatchMe struct {
	k KhronosConn
	w *Watcher

	lastNodeMetrics atomic.Pointer[v1beta1.NodeMetricsList]
}

func (n *NodeWatchMe) getMetricsForNode(node *corev1.Node) map[string]string {
	metricsExtra := map[string]string{}
	lastNodeMetrics := n.lastNodeMetrics.Load()
	if lastNodeMetrics == nil {
		return metricsExtra
	}
	for _, nodeMetrics := range lastNodeMetrics.Items {
		if nodeMetrics.Name == node.Name {
			cpuUsage := nodeMetrics.Usage[corev1.ResourceCPU]
			memUsage := nodeMetrics.Usage[corev1.ResourceMemory]

			cpuCapacity := node.Status.Capacity[corev1.ResourceCPU]
			memCapacity := node.Status.Capacity[corev1.ResourceMemory]

			cpuPercentage := calculatePercentage(cpuUsage.MilliValue(), cpuCapacity.MilliValue())
			memPercentage := calculatePercentage(memUsage.Value(), memCapacity.Value())

			metricsExtra[node.Name] = fmt.Sprintf("%s %s", renderProgressBar("CPU", cpuPercentage), renderProgressBar("Mem", memPercentage))

			return metricsExtra
		}
	}
	return metricsExtra
}

func (n *NodeWatchMe) updateResourceMetrics(resource Resource) {
	node := resource.Object.(*corev1.Node)

	metricsExtra := n.getMetricsForNode(node)
	if len(metricsExtra) > 0 {
		resource = resource.SetExtraKV("Metrics", metricsExtra)
		resource.Timestamp = time.Now()
		n.w.Update(resource)
	}
}

func (n *NodeWatchMe) Tick() {

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	m, err := n.k.mc.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	n.lastNodeMetrics.Store(m)

	// // Get the current resources
	resources := n.w.GetStateAtTime(time.Now(), "Node", "")
	for _, resource := range resources {
		n.updateResourceMetrics(resource)
	}
}

func (n *NodeWatchMe) Kind() string {
	return "Node"
}

func (n *NodeWatchMe) Renderer() ResourceRenderer {
	return NodeRenderer{n}
}

func (n *NodeWatchMe) convert(obj runtime.Object) *corev1.Node {
	ret, ok := obj.(*corev1.Node)
	if !ok {
		return nil
	}
	return ret
}

func (n *NodeWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n *NodeWatchMe) getExtra(node *corev1.Node) map[string]any {
	extra := map[string]any{}
	extra["Metrics"] = n.getMetricsForNode(node)

	// Calculate the running time
	startTime := node.ObjectMeta.CreationTimestamp
	extra["StartTime"] = startTime.Time

	return extra
}

func (n *NodeWatchMe) update(obj runtime.Object) *Resource {
	r := n.Modified(obj)
	return &r
}

func (n *NodeWatchMe) Add(obj runtime.Object) Resource {
	node := n.convert(obj)
	return NewResource(node.ObjectMeta.CreationTimestamp.Time, n.Kind(), node.Namespace, node.Name, node, n.Renderer()).SetExtra(n.getExtra(node))
}
func (n *NodeWatchMe) Modified(obj runtime.Object) Resource {
	node := n.convert(obj)
	return NewResource(time.Now(), n.Kind(), node.Namespace, node.Name, node, n.Renderer()).SetExtra(n.getExtra(node))
}
func (n *NodeWatchMe) Del(obj runtime.Object) Resource {
	node := n.convert(obj)
	return NewResource(node.ObjectMeta.DeletionTimestamp.Time, n.Kind(), node.Namespace, node.Name, node, n.Renderer()).SetExtra(n.getExtra(node))

}

func watchForNodes(watcher *Watcher, k KhronosConn) {
	fmt.Println("Watching nodes . . .")
	watchChan, err := k.client.CoreV1().Nodes().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.watchEvents(watchChan.ResultChan(), &NodeWatchMe{k: k, w: watcher})
}
