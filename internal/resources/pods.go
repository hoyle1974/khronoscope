package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type PodRenderer struct {
	dao DAO
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
			out = append(out, fmt.Sprintf("   %s - %s : %s", containerName, containerInfo.Image, ""))
		}
		s, _ := misc.PrettyPrintYAMLFromJSON(resource.RawJSON)
		out = append(out, strings.Split(s, "\n")...)

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

type Pod struct {
	Spec struct {
		Containers []struct {
			Image     string `json:"image"`
			Name      string `json:"name"`
			Resources struct {
				Limits struct {
					CPU    string `json:"cpu"`
					Memory string `json:"memory"`
				} `json:"limits"`
			} `json:"resources"`
		} `json:"containers"`
		NodeName string `json:"nodeName"`
	} `json:"spec"`
	Status struct {
		Phase string `json:"phase"`
	} `json:"status"`
}

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
	Phase      string
	NodeName   string
	Metrics    map[string]PodMetric
	Uptime     time.Duration
	StartTime  serializable.Time
	Containers map[string]ContainerInfo
	Logs       []string
	Logging    []string
}

func (r PodExtra) GetValue(key string) any { return nil }

func (p PodExtra) Copy() Copyable {
	return PodExtra{
		Phase:      p.Phase,
		NodeName:   p.NodeName,
		Metrics:    misc.DeepCopyMap(p.Metrics),
		Uptime:     p.Uptime,
		StartTime:  p.StartTime,
		Containers: misc.DeepCopyMap(p.Containers),
		Logs:       misc.DeepCopyArray(p.Logs),
		Logging:    misc.DeepCopyArray(p.Logging),
	}
}

func getPodExtra(resource Resource) PodExtra {
	var extra PodExtra
	if resource.Extra != nil {
		extra = resource.Extra.Copy().(PodExtra)
	} else {
		var pod Pod
		extra.Containers = map[string]ContainerInfo{}
		extra.Metrics = map[string]PodMetric{}
		if err := json.Unmarshal([]byte(resource.RawJSON), &pod); err == nil {
			extra.Phase = pod.Status.Phase
			extra.NodeName = pod.Spec.NodeName
			for _, container := range pod.Spec.Containers {
				cpuLimit, _ := strconv.Atoi(container.Resources.Limits.CPU)
				memory, _ := misc.ParseMemory(container.Resources.Limits.Memory)

				extra.Containers[container.Name] = ContainerInfo{
					Image:       container.Image,
					MemoryLimit: memory,
					CPULimit:    int64(cpuLimit * 1000),
				}
			}
		}
	}

	return extra
}

var lastPodMetrics atomic.Pointer[v1beta1.PodMetricsList]

func podTicker(dao DAO, metricsClient *metrics.Clientset) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	m, err := metricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	lastPodMetrics.Store(m)

	// // Get the current resources
	resources := dao.GetResourcesAt(time.Now(), "Pod", "")
	for _, resource := range resources {
		updatePodResourceMetrics(dao, resource)
	}
}

func calculatePercentage(usage int64, limit int64) float64 {
	if limit == 0 {
		return 0
	}
	return (float64(usage) / float64(limit)) * 100
}

func getPodMetricsForPod(resource Resource) map[string]PodMetric {
	extra := getPodExtra(resource)

	metricsExtra := map[string]PodMetric{}
	lastPodMetrics := lastPodMetrics.Load()
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

func updatePodResourceMetrics(dao DAO, resource Resource) {
	extra := getPodExtra(resource).Copy().(PodExtra)

	metricsExtra := getPodMetricsForPod(resource)
	if len(metricsExtra) > 0 {
		extra.Metrics = metricsExtra
	}
	extra.Uptime = time.Since(extra.StartTime.Time).Truncate(time.Second)
	resource.Timestamp = serializable.Time{Time: time.Now()}
	resource.Extra = extra
	dao.UpdateResource(resource)
}
