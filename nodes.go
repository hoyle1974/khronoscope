package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type NodeRenderer struct {
	n *NodeWatcher
}

func describeNode(node *corev1.Node) []string {
	out := []string{}

	out = append(out, fmt.Sprintf("Roles: %s", getNodeRoles(node)))

	out = append(out, RenderMapOfStrings("Labels:", node.GetLabels())...)
	out = append(out, RenderMapOfStrings("Annotations:", node.GetAnnotations())...)

	out = append(out, "Capacity:")
	for resource, quantity := range NewMapRangeFunc(node.Status.Capacity) {
		out = append(out, fmt.Sprintf("  %s: %s", resource, quantity.String()))
	}
	out = append(out, "Allocatable:")
	for resource, quantity := range NewMapRangeFunc(node.Status.Allocatable) {
		out = append(out, fmt.Sprintf("  %s: %s", resource, quantity.String()))
	}

	// print the relevant system information
	out = append(out, fmt.Sprintf("Node Name: %s", node.Name))
	out = append(out, fmt.Sprintf("Machine ID: %s", node.Status.NodeInfo.MachineID))
	out = append(out, fmt.Sprintf("System UUID: %s", node.Status.NodeInfo.SystemUUID))
	out = append(out, fmt.Sprintf("Boot ID: %s", node.Status.NodeInfo.BootID))
	out = append(out, fmt.Sprintf("Kernel Version: %s", node.Status.NodeInfo.KernelVersion))
	out = append(out, fmt.Sprintf("OS Image: %s", node.Status.NodeInfo.OSImage))
	out = append(out, fmt.Sprintf("Container Runtime Version: %s", node.Status.NodeInfo.ContainerRuntimeVersion))
	out = append(out, fmt.Sprintf("Kubelet Version: %s", node.Status.NodeInfo.KubeletVersion))
	out = append(out, fmt.Sprintf("Kube-Proxy Version: %s", node.Status.NodeInfo.KubeProxyVersion))
	out = append(out, fmt.Sprintf("Operating System: %s", node.Status.NodeInfo.OperatingSystem))
	out = append(out, fmt.Sprintf("Architecture: %s", node.Status.NodeInfo.Architecture))

	out = append(out, "Addresses:")
	for _, address := range node.Status.Addresses {
		out = append(out, fmt.Sprintf("  %s: %s", address.Type, address.Address))
	}

	out = append(out, "Images:")
	for _, image := range node.Status.Images {
		out = append(out, fmt.Sprintf("  - Names: %s", strings.Join(image.Names, ", ")))
		out = append(out, fmt.Sprintf("    Size: %d bytes", image.SizeBytes))
	}

	out = append(out, "Conditions:")
	for _, condition := range node.Status.Conditions {
		out = append(out, fmt.Sprintf("  Type: %s", condition.Type))
		out = append(out, fmt.Sprintf("  Status: %s", condition.Status))
		out = append(out, fmt.Sprintf("  LastHeartbeatTime: %s", condition.LastHeartbeatTime.Time.Format(time.RFC3339)))
		out = append(out, fmt.Sprintf("  LastTransitionTime: %s", condition.LastTransitionTime.Time.Format(time.RFC3339)))
		out = append(out, fmt.Sprintf("  Reason: %s", condition.Reason))
		out = append(out, fmt.Sprintf("  Message: %s", condition.Message))
		out = append(out, fmt.Sprintf(""))
	}

	return out
}

// Helper function to get node roles
func getNodeRoles(node *corev1.Node) string {
	roles := []string{}
	for label := range node.Labels {
		if strings.HasPrefix(label, "kubernetes.io/role/") {
			role := strings.TrimPrefix(label, "kubernetes.io/role/")
			roles = append(roles, role)
		}
	}
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return "<none>"
	}
	sort.Strings(roles)
	return strings.Join(roles, ", ")
}

func getPodsOnNode(client kubernetes.Interface, nodeName string) ([]corev1.Pod, error) {
	listOptions := metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	}
	pods, err := client.CoreV1().Pods("").List(context.TODO(), listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	return pods.Items, nil
}

func (r NodeRenderer) Render(resource Resource, details bool) []string {
	extra := resource.GetExtra()

	if details {
		node := resource.Object.(*corev1.Node)
		ret := []string{}
		ret = append(ret, "Name: "+resource.Name)

		if rt, ok := extra["StartTime"]; ok {
			ret = append(ret, fmt.Sprintf(" Uptime:%v", rt))
		}
		if e, ok := extra["Metrics"]; ok {
			ret = append(ret, "")
			ret = append(ret, "Metrics: ")
			m := e.(map[string]string)
			ret = append(ret, fmt.Sprintf("%v", m[resource.Name]))
		}

		// ret = append(ret, RenderMapOfStrings("Labels:", node.GetLabels())...)
		// ret = append(ret, RenderMapOfStrings("Annotations:", node.GetAnnotations())...)

		if p, ok := extra["Pods"]; ok {
			ret = append(ret, "")
			ret = append(ret, "Pods: ")
			pods := p.([]corev1.Pod)

			if pm, ok := extra["PodMetrics"]; ok {
				podMetrics := pm.(map[string]map[string]PodMetric)

				for _, pod := range pods {
					var cpu float64 = 0
					var mem float64 = 0
					bar := ""

					if podMetric, ok := podMetrics[string(pod.UID)]; ok {
						for _, v := range podMetric {
							cpu += v.cpuPercentage
							mem += v.memoryPercentage
						}
						if len(podMetric) > 0 {
							cpu /= float64(len(podMetric))
							mem /= float64(len(podMetric))
							bar = fmt.Sprintf("%s %s : ", renderProgressBar("CPU", cpu), renderProgressBar("Mem", mem))
						} else {
							bar = strings.Repeat(" ", 29) + " : "
						}
					}

					nn := bar + pod.Namespace + "/" + pod.Name
					ret = append(ret, "   "+nn)
				}
			} else {
				for _, pod := range pods {
					nn := pod.Namespace + "/" + pod.Name
					ret = append(ret, "   "+nn)
				}

			}

		}

		ret = append(ret, describeNode(node)...)

		return ret
	}

	out := ""
	e, ok := extra["Metrics"]
	if ok {
		m := e.(map[string]string)
		out += fmt.Sprintf("%v", m[resource.Name])
	}
	out += " " + resource.Name

	rt, ok := extra["StartTime"]
	if ok {
		out += fmt.Sprintf(" %v", rt)
	}

	return []string{out}
}

type NodeWatcher struct {
	k   KhronosConn
	w   *K8sWatcher
	pwm *PodWatcher

	lastNodeMetrics atomic.Pointer[v1beta1.NodeMetricsList]
}

func (n *NodeWatcher) getMetricsForNode(node *corev1.Node) map[string]string {
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

func (n *NodeWatcher) updateResourceMetrics(resource Resource) {
	node := resource.Object.(*corev1.Node)

	metricsExtra := n.getMetricsForNode(node)
	if len(metricsExtra) > 0 {
		resource = resource.SetExtraKV("Metrics", metricsExtra)
		resource = resource.SetExtraKV("StartTime", time.Since(node.ObjectMeta.CreationTimestamp.Time).Truncate(time.Second))
		resource.Timestamp = time.Now()
	}

	pods, err := getPodsOnNode(n.k.client, node.Name)
	if err == nil {
		resource = resource.SetExtraKV("Pods", pods)

		podMetrics := map[string]map[string]PodMetric{}
		for _, pod := range pods {
			podMetrics[string(pod.UID)] = n.pwm.getPodMetricsForPod(&pod)
		}
		resource = resource.SetExtraKV("PodMetrics", podMetrics)
	}

	n.w.Update(resource)

}

func (n *NodeWatcher) Tick() {

	ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
	m, err := n.k.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
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

func (n *NodeWatcher) Kind() string {
	return "Node"
}

func (n *NodeWatcher) Renderer() ResourceRenderer {
	return NodeRenderer{n}
}

func (n *NodeWatcher) convert(obj runtime.Object) *corev1.Node {
	ret, ok := obj.(*corev1.Node)
	if !ok {
		return nil
	}
	return ret
}

func (n *NodeWatcher) getExtra(node *corev1.Node) map[string]any {
	extra := map[string]any{}
	extra["Metrics"] = n.getMetricsForNode(node)

	// Calculate the running time
	startTime := node.ObjectMeta.CreationTimestamp
	extra["StartTime"] = time.Since(startTime.Time).Truncate(time.Second)
	return extra
}

func (n *NodeWatcher) ToResource(obj runtime.Object) Resource {
	node := n.convert(obj)
	return NewK8sResource(n.Kind(), node, n.Renderer()).SetExtra(n.getExtra(node))
}

func watchForNodes(watcher *K8sWatcher, k KhronosConn, pwm *PodWatcher) *NodeWatcher {
	watchChan, err := k.client.CoreV1().Nodes().Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	w := &NodeWatcher{k: k, w: watcher, pwm: pwm}

	go watcher.registerEventWatcher(watchChan.ResultChan(), w)

	return w
}
