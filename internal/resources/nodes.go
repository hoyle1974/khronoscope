package resources

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	"github.com/hoyle1974/khronoscope/internal/ui"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type NodeExtra struct {
	Metrics               map[string]string
	NodeCreationTimestamp time.Time
	CPUCapacity           int64
	MemCapacity           int64
	Uptime                time.Duration
	PodMetrics            map[string]map[string]PodMetric
}

func (n NodeExtra) Copy() NodeExtra {
	return NodeExtra{
		Metrics:               misc.DeepCopyMap(n.Metrics),
		NodeCreationTimestamp: n.NodeCreationTimestamp,
		CPUCapacity:           n.CPUCapacity,
		MemCapacity:           n.MemCapacity,
		Uptime:                n.Uptime,
		PodMetrics:            misc.DeepCopyMap(n.PodMetrics),
	}
}

type NodeRenderer struct {
	d DAO
}

func (r NodeRenderer) Render(resource Resource, details bool) []string {
	if resource.Extra == nil {
		return []string{}
	}

	extra := resource.Extra.(NodeExtra)

	if details {
		// node := obj.(*corev1.Node)
		ret := []string{}
		ret = append(ret, "Name: "+resource.Name)

		ret = append(ret, fmt.Sprintf(" Uptime:%v", extra.Uptime))

		ret = append(ret, "")
		ret = append(ret, "Metrics: ")
		ret = append(ret, fmt.Sprintf("%v", extra.Metrics[resource.Name]))

		ret = append(ret, "Pods: ")
		for podName, podMetrics := range misc.Range(extra.PodMetrics) {
			var cpu float64 = 0
			var mem float64 = 0
			bar := ""

			for _, podMetric := range podMetrics {
				cpu += podMetric.CPUPercentage
				mem += podMetric.MemoryPercentage
			}
			if len(podMetrics) > 0 {
				cpu /= float64(len(podMetrics))
				mem /= float64(len(podMetrics))
				bar = fmt.Sprintf("%s %s : ", misc.RenderProgressBar("CPU", cpu), misc.RenderProgressBar("Mem", mem))
			} else {
				bar = strings.Repeat(" ", 29) + " : "
			}
			nn := bar + podName
			ret = append(ret, "   "+nn)

		}

		ret = append(ret, resource.Details...)

		return ret
	}

	out := fmt.Sprintf("%v", extra.Metrics[resource.Name])
	out += " " + resource.Name
	out += fmt.Sprintf(" %v", extra.Uptime)

	return []string{out}
}

type NodeWatcher struct {
	k conn.KhronosConn
	// w   *K8sWatcher
	d   DAO
	pwm *PodWatcher

	lastNodeMetrics atomic.Pointer[v1beta1.NodeMetricsList]
}

func (n *NodeWatcher) getMetricsForNode(resource Resource) map[string]string {
	e := resource.Extra.(NodeExtra)

	metricsExtra := map[string]string{}
	lastNodeMetrics := n.lastNodeMetrics.Load()
	if lastNodeMetrics == nil {
		return metricsExtra
	}
	for _, nodeMetrics := range lastNodeMetrics.Items {
		if nodeMetrics.Name == resource.Name {
			cpuUsage := nodeMetrics.Usage[corev1.ResourceCPU]
			memUsage := nodeMetrics.Usage[corev1.ResourceMemory]

			cpuPercentage := calculatePercentage(cpuUsage.MilliValue(), e.CPUCapacity)
			memPercentage := calculatePercentage(memUsage.Value(), e.MemCapacity)

			metricsExtra[resource.Name] = fmt.Sprintf("%s %s", misc.RenderProgressBar("CPU", cpuPercentage), misc.RenderProgressBar("Mem", memPercentage))

			return metricsExtra
		}
	}
	return metricsExtra
}

func (n *NodeWatcher) updateResourceMetrics(resource Resource) {

	e := resource.Extra.(NodeExtra).Copy()
	resource.Timestamp = serializable.Time{Time: time.Now()}

	metricsExtra := n.getMetricsForNode(resource)
	if len(metricsExtra) > 0 {
		e.Metrics = metricsExtra
		e.Uptime = time.Since(e.NodeCreationTimestamp).Truncate(time.Second)
	}

	// Find pods on node
	resources := n.d.GetResourcesAt(resource.Timestamp.Time, "Pod", "")
	e.PodMetrics = map[string]map[string]PodMetric{}
	for _, podResource := range resources {
		if podResource.Extra.(PodExtra).Node == resource.Name {
			podMetrics := n.pwm.getPodMetricsForPod(podResource)
			e.PodMetrics[podResource.Namespace+"/"+podResource.Name] = podMetrics
		}
	}

	resource.Extra = e

	n.d.UpdateResource(resource)

}

func (n *NodeWatcher) Tick() {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	m, err := n.k.MetricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	n.lastNodeMetrics.Store(m)

	// // Get the current resources
	resources := n.d.GetResourcesAt(time.Now(), "Node", "")
	for _, resource := range resources {
		n.updateResourceMetrics(resource)
	}
}

func (n *NodeWatcher) Kind() string {
	return "Node"
}

func (n *NodeWatcher) Renderer() ResourceRenderer {
	return NodeRenderer{n.d}
}

func (n *NodeWatcher) convert(obj runtime.Object) *corev1.Node {
	ret, ok := obj.(*corev1.Node)
	if !ok {
		return nil
	}
	return ret
}

func (n *NodeWatcher) ToResource(obj runtime.Object) Resource {
	node := n.convert(obj)

	cpuCapacity := node.Status.Capacity[corev1.ResourceCPU]
	memCapacity := node.Status.Capacity[corev1.ResourceMemory]

	extra := NodeExtra{
		CPUCapacity:           cpuCapacity.MilliValue(),
		MemCapacity:           memCapacity.Value(),
		NodeCreationTimestamp: node.CreationTimestamp.Time,
		Uptime:                time.Since(node.CreationTimestamp.Time).Truncate(time.Second),
	}

	return NewK8sResource(n.Kind(), node, ui.FormatNodeDetails(node), extra)
}

func watchForNodes(watcher *K8sWatcher, k conn.KhronosConn, d DAO, pwm *PodWatcher) (*NodeWatcher, error) {
	watchChan, err := k.Client.CoreV1().Nodes().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	w := &NodeWatcher{k: k, d: d, pwm: pwm}

	go watcher.registerEventWatcher(watchChan.ResultChan(), w)

	return w, nil
}
