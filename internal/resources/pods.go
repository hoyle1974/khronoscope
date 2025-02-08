package resources

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/misc/format"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type PodMetric struct {
	CPUPercentage    float64
	MemoryPercentage float64
}

type ContainerInfo struct {
	Image       string
	CPULimit    int64
	MemoryLimit int64
}

type PodExtra struct {
	Phase       string
	Node        string
	Metrics     map[string]PodMetric
	Uptime      time.Duration
	StartTime   serializable.Time
	Containers  map[string]ContainerInfo
	Labels      []string
	Annotations []string
	Logs        []string
	Logging     []string
}

func (r PodExtra) GetValue(key string) any { return nil }

func (p PodExtra) Copy() Copyable {
	return PodExtra{
		Phase:       p.Phase,
		Node:        p.Node,
		Metrics:     misc.DeepCopyMap(p.Metrics),
		Uptime:      p.Uptime,
		StartTime:   p.StartTime,
		Containers:  misc.DeepCopyMap(p.Containers),
		Labels:      p.Labels,
		Annotations: p.Annotations,
		Logs:        misc.DeepCopyArray(p.Logs),
		Logging:     misc.DeepCopyArray(p.Logging),
	}
}

type PodRenderer struct {
}

func (r PodRenderer) Render(resource Resource, details bool) []string {
	if resource.Extra == nil {
		return []string{}
	}

	extra := resource.Extra.(PodExtra)
	out := []string{}
	s := ""

	if details {
		out = append(out, fmt.Sprintf("Name: %s", resource.Name))
		out = append(out, fmt.Sprintf("Namespace: %s", resource.Namespace))

		s += fmt.Sprintf("Phase: %v\n", extra.Phase)
		s += fmt.Sprintf("Node: %s\n", extra.Node)

		s += fmt.Sprintf("Uptime: %v\n", extra.Uptime)
		out = append(out, s)

		m := extra.Metrics
		if len(m) > 0 {
			out = append(out, "containers:")
			sortedKeys := slices.Sorted(maps.Keys(m))
			for _, k := range sortedKeys {
				bar := fmt.Sprintf("%s %s", misc.RenderProgressBar("CPU", m[k].CPUPercentage), misc.RenderProgressBar("Mem", m[k].MemoryPercentage))
				out = append(out, fmt.Sprintf("   %v - %v", bar, k))
			}
		}

		// Print container information
		out = append(out, "Containers:")
		for containerName, containerInfo := range extra.Containers {
			out = append(out, fmt.Sprintf("   %s - %s : %s", containerName, containerInfo.Image, "" /* getContainerState(containerName)*/))
		}
		// for _, container := range pod.Spec.Containers {
		// 	out = append(out, fmt.Sprintf("   %s - %s : %s", container.Name, container.Image, getContainerState(container.Name)))
		// }

		out = append(out, extra.Labels...)
		out = append(out, extra.Annotations...)
		out = append(out, resource.Details...)

	} else {
		p := fmt.Sprintf("%v", extra.Phase)

		switch p {
		case "Pending":
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#AAAAAA"))
			s += fmt.Sprintf(" [%s]", style.Render(p))
		case "Failed":
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF4444"))
			s += fmt.Sprintf(" [%s]", style.Render(p))
		case "Unknown":
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF"))
			s += fmt.Sprintf(" [%s]", style.Render(p))
		case "Running":
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
			s += fmt.Sprintf(" [%s]", style.Render(p))
		case "Succeeded":
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#0000FF"))
			s += fmt.Sprintf(" [%s]", style.Render(p))
		default:
			style := lipgloss.NewStyle()
			s += fmt.Sprintf(" [%s]", style.Render(p))
		}
		s += " " + resource.Name

		podMetric := extra.Metrics
		var cpu float64
		var mem float64
		bar := ""

		for _, v := range podMetric {
			cpu += v.CPUPercentage
			mem += v.MemoryPercentage
		}
		if len(podMetric) > 0 {
			cpu /= float64(len(podMetric))
			mem /= float64(len(podMetric))
			bar = fmt.Sprintf("%s %s : ", misc.RenderProgressBar("CPU", cpu), misc.RenderProgressBar("Mem", mem))
		} else {
			bar = strings.Repeat(" ", 29) + " : "
		}

		s = bar + s
		s += fmt.Sprintf(" %v", extra.Uptime)

		out = append(out, s)
	}

	return out
}

type PodWatcher struct {
	k  conn.KhronosConn
	d  DAO
	lc *LogCollector

	lastPodMetrics atomic.Pointer[v1beta1.PodMetricsList]
}

func calculatePercentage(usage int64, limit int64) float64 {
	if limit == 0 {
		return 0
	}
	return (float64(usage) / float64(limit)) * 100
}

func (n *PodWatcher) getPodMetricsForPod(resource Resource) map[string]PodMetric {

	extra := resource.Extra.(PodExtra)
	metricsExtra := map[string]PodMetric{}
	lastPodMetrics := n.lastPodMetrics.Load()
	if lastPodMetrics == nil {
		return metricsExtra
	}
	for _, podMetrics := range lastPodMetrics.Items {
		if podMetrics.Namespace == resource.Namespace && podMetrics.Name == resource.Name {
			for containerName, limits := range extra.Containers {
				for _, containerMetric := range podMetrics.Containers {
					if containerName == containerMetric.Name {
						cpuUsage := containerMetric.Usage[corev1.ResourceCPU]
						memoryUsage := containerMetric.Usage[corev1.ResourceMemory]

						cpuPercentage := calculatePercentage(cpuUsage.MilliValue(), limits.CPULimit)
						memoryPercentage := calculatePercentage(memoryUsage.Value(), limits.MemoryLimit)

						metricsExtra[containerName] = PodMetric{
							CPUPercentage:    cpuPercentage,
							MemoryPercentage: memoryPercentage,
						}
					}
				}
			}
			return metricsExtra
		}
	}
	return metricsExtra
}

func (n *PodWatcher) updateResourceMetrics(resource Resource) {
	extra := resource.Extra.Copy().(PodExtra)

	metricsExtra := n.getPodMetricsForPod(resource)
	if len(metricsExtra) > 0 {
		extra.Metrics = metricsExtra
	}
	extra.Uptime = time.Since(extra.StartTime.Time).Truncate(time.Second)
	resource.Timestamp = serializable.Time{Time: time.Now()}
	resource.Extra = extra
	n.d.UpdateResource(resource)
}

func (n *PodWatcher) Tick() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	m, err := n.k.MetricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	n.lastPodMetrics.Store(m)

	// // Get the current resources
	resources := n.d.GetResourcesAt(time.Now(), "Pod", "")
	for _, resource := range resources {
		n.updateResourceMetrics(resource)
	}
}

func (n *PodWatcher) Kind() string {
	return "Pod"
}

func (n *PodWatcher) Renderer() ResourceRenderer {
	return PodRenderer{}
}

func (n *PodWatcher) convert(obj runtime.Object) *corev1.Pod {
	ret, ok := obj.(*corev1.Pod)
	if !ok {
		return nil
	}
	return ret
}

func (n *PodWatcher) ToResource(obj runtime.Object) Resource {
	pod := n.convert(obj)

	containerLimits := map[string]ContainerInfo{}
	for _, container := range pod.Spec.Containers {
		cpuLimit := container.Resources.Limits[corev1.ResourceCPU]
		memoryLimit := container.Resources.Limits[corev1.ResourceMemory]

		containerLimits[container.Name] = ContainerInfo{
			CPULimit:    cpuLimit.MilliValue(),
			MemoryLimit: memoryLimit.Value(),
			Image:       container.Image,
		}
	}

	extra := PodExtra{
		Phase:       fmt.Sprintf("%v", pod.Status.Phase),
		Node:        pod.Spec.NodeName,
		Containers:  containerLimits,
		StartTime:   serializable.Time{Time: pod.CreationTimestamp.Time},
		Labels:      misc.RenderMapOfStrings("Labels:", pod.GetLabels()),
		Annotations: misc.RenderMapOfStrings("Annotations:", pod.GetAnnotations()),
	}

	return NewK8sResource(n.Kind(), pod, format.FormatPodDetails(pod), extra)
}

func watchForPods(watcher *K8sWatcher, k conn.KhronosConn, d DAO, lc *LogCollector) (*PodWatcher, error) {
	watchChan, err := k.Client.CoreV1().Pods("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	w := &PodWatcher{k: k, d: d, lc: lc}
	go watcher.registerEventWatcher(watchChan.ResultChan(), w)

	return w, nil
}
