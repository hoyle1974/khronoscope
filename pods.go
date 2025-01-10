package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

func grommet(is bool) string {
	if !is {
		return "├"
	}
	return "└"
}
func grommet2(is bool) string {
	if !is {
		return "│"
	}
	return " "
}

func getPodLogs(client kubernetes.Interface, namespace, podName string) (string, error) {
	return "", nil

	lines := int64(15)
	req := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		TailLines: &lines,
	})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", fmt.Errorf("error opening stream: %w", err)
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", fmt.Errorf("error copying logs: %w", err)
	}

	return buf.String(), nil
}

type PodRenderer struct {
	n *PodWatchMe
}

func (r PodRenderer) Render(resource Resource, details bool) []string {
	extra := resource.GetExtra()
	out := []string{}
	s := ""

	if details {
		pod := resource.Object.(*corev1.Pod)

		out = append(out, fmt.Sprintf("Name: %s", pod.Name))
		out = append(out, fmt.Sprintf("Namespace: %s", pod.Namespace))

		phase, ok := extra["Phase"]
		if ok {
			s += fmt.Sprintf("Phase: %v\n", phase)
		}
		node, ok := extra["Node"]
		if ok {
			s += fmt.Sprintf("Node: %s\n", node)
		}
		rt, ok := extra["StartTime"]
		if ok {
			s += fmt.Sprintf("Uptime: %v\n", rt)
		}
		out = append(out, s)

		e, ok := extra["Metrics"]
		if ok {
			m := e.(map[string]PodMetric)
			if len(m) > 0 {
				out = append(out, "containers:")
				sortedKeys := slices.Sorted(maps.Keys(m))
				for _, k := range sortedKeys {
					bar := fmt.Sprintf("%s %s", renderProgressBar("CPU", m[k].cpuPercentage), renderProgressBar("Mem", m[k].memoryPercentage))
					out = append(out, fmt.Sprintf("   %v - %v", bar, k))
				}
			}
		}

		out = append(out, fmt.Sprintf("Generation: %v", pod.GetGeneration()))

		getContainerState := func(cname string) string {
			for _, status := range pod.Status.ContainerStatuses {
				if status.Name == cname {
					if status.State.Waiting != nil {
						return "Waiting: " + status.State.Waiting.Reason
					}
					if status.State.Running != nil {
						return fmt.Sprintf("Running")
					}
					if status.State.Terminated != nil {
						return "Terminated: " + status.State.Terminated.Reason
					}
					return "unknown"
				}
			}
			return ""
		}

		// Print container information
		out = append(out, fmt.Sprintf("Containers:"))
		for _, container := range pod.Spec.Containers {
			out = append(out, fmt.Sprintf("   %s - %s : %s", container.Name, container.Image, getContainerState(container.Name)))
		}

		out = append(out, RenderMapOfStrings("Labels:", pod.GetLabels())...)
		out = append(out, RenderMapOfStrings("Annotations:", pod.GetAnnotations())...)

		logs, err := getPodLogs(r.n.k.client, pod.Namespace, pod.Name)
		if err == nil {
			lines := strings.Split(logs, "\n")
			if len(lines) > 10 {
				lines = lines[len(lines)-10:]
			}
			out = append(out, lines...)
		} else {
			out = append(out, fmt.Sprintf("%v", err))
		}
	} else {
		phase, ok := extra["Phase"]
		if ok {
			p := fmt.Sprintf("%v", phase)

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
		}
		s += " " + resource.Name

		e, ok := extra["Metrics"]
		if ok {
			podMetric := e.(map[string]PodMetric)
			var cpu float64
			var mem float64
			bar := ""

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

			s = bar + s
		}
		rt, ok := extra["StartTime"]
		if ok {
			s += fmt.Sprintf(" %v", rt)
		}
		out = append(out, s)
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

type PodMetric struct {
	cpuPercentage    float64
	memoryPercentage float64
}

func (n *PodWatchMe) getPodMetricsForPod(pod *corev1.Pod) map[string]PodMetric {
	metricsExtra := map[string]PodMetric{}
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

						metricsExtra[container.Name] = PodMetric{
							cpuPercentage:    cpuPercentage,
							memoryPercentage: memoryPercentage,
						}
					}
				}
			}
			return metricsExtra
		}
	}
	return metricsExtra
}

/*
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

						metricsExtra[container.Name] = fmt.Sprintf("%s %s", renderProgressBar("CPU", cpuPercentage), renderProgressBar("Mem", memoryPercentage))
					}
				}
			}
			return metricsExtra
		}
	}
	return metricsExtra
}
*/

func (n *PodWatchMe) updateResourceMetrics(resource Resource) {
	pod := resource.Object.(*corev1.Pod)

	metricsExtra := n.getPodMetricsForPod(pod)
	if len(metricsExtra) > 0 {
		resource = resource.SetExtraKV("Metrics", metricsExtra)
		if pod.Status.StartTime != nil {
			resource = resource.SetExtraKV("StartTime", time.Since(pod.Status.StartTime.Time).Truncate(time.Second))
		}

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
	extra["Metrics"] = n.getPodMetricsForPod(pod)

	// Calculate the running time
	startTime := pod.Status.StartTime
	if startTime != nil {
		extra["StartTime"] = time.Since(pod.Status.StartTime.Time).Truncate(time.Second)
	}

	return extra
}

func (n *PodWatchMe) Add(obj runtime.Object) Resource {
	pod := n.convert(obj)
	return NewResource(string(pod.ObjectMeta.GetUID()), pod.ObjectMeta.CreationTimestamp.Time, n.Kind(), pod.Namespace, pod.Name, pod, n.Renderer()).SetExtra(n.getExtra(pod))
}
func (n *PodWatchMe) Modified(obj runtime.Object) Resource {
	pod := n.convert(obj)
	return NewResource(string(pod.ObjectMeta.GetUID()), time.Now(), n.Kind(), pod.Namespace, pod.Name, pod, n.Renderer()).SetExtra(n.getExtra(pod))
}
func (n *PodWatchMe) Del(obj runtime.Object) Resource {
	pod := n.convert(obj)
	r := NewResource(string(pod.ObjectMeta.GetUID()), time.Now() /*pod.DeletionTimestamp.Time*/, n.Kind(), pod.Namespace, pod.Name, pod, n.Renderer()).SetExtra(n.getExtra(pod))
	return r
}

func watchForPods(watcher *Watcher, k KhronosConn) *PodWatchMe {
	fmt.Println("Watching pods . . .")
	watchChan, err := k.client.CoreV1().Pods("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	w := &PodWatchMe{k: k, w: watcher}
	go watcher.watchEvents(watchChan.ResultChan(), w)

	return w
}
