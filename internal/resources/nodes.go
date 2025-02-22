package resources

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

type NodeExtra struct {
	NodeMetrics           map[string]string
	NodeCreationTimestamp time.Time
	CPUCapacity           int64
	MemCapacity           int64
	Uptime                time.Duration
	PodMetrics            map[string]map[string]PodMetric
}

func (n NodeExtra) Copy() Copyable {
	return NodeExtra{
		NodeMetrics:           misc.DeepCopyMap(n.NodeMetrics),
		PodMetrics:            misc.DeepCopyMap(n.PodMetrics),
		NodeCreationTimestamp: n.NodeCreationTimestamp,
		CPUCapacity:           n.CPUCapacity,
		MemCapacity:           n.MemCapacity,
		Uptime:                n.Uptime,
	}
}

type NodeRenderer struct {
	d DAO
}

func (r NodeRenderer) Render(resource Resource, details bool) []string {
	extra := getNodeExtra(resource)

	if details {
		ret := []string{}

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

		s, _ := misc.PrettyPrintYAMLFromJSON(resource.RawJSON)
		ret = append(ret, strings.Split(s, "\n")...)

		return ret
	}

	out := fmt.Sprintf("%v", extra.NodeMetrics[resource.Name])
	out += " " + resource.Name
	out += fmt.Sprintf(" %v", extra.Uptime)

	return []string{out}

}

func updateResourceMetrics(podWatcher *PodWatcher, dao DAO, resource Resource) {
	e := getNodeExtra(resource)

	resource.Timestamp = serializable.Time{Time: time.Now()}

	metricsExtra := getMetricsForNode(resource)
	if len(metricsExtra) > 0 {
		e.NodeMetrics = metricsExtra
		e.Uptime = time.Since(e.NodeCreationTimestamp).Truncate(time.Second)
	}

	// Find pods on node
	resources := dao.GetResourcesAt(resource.Timestamp.Time, "Pod", "")
	e.PodMetrics = map[string]map[string]PodMetric{}
	for _, podResource := range resources {
		if podResource.Extra.(PodExtra).Node == resource.Name {
			podMetrics := podWatcher.getPodMetricsForPod(podResource)
			e.PodMetrics[podResource.Namespace+"/"+podResource.Name] = podMetrics
		}
	}

	resource.Extra = e

	dao.UpdateResource(resource)

}

var lastNodeMetrics atomic.Pointer[v1beta1.NodeMetricsList]

type NodeMetadata struct {
	CreationTimestamp string `json:"creationTimestamp"`
}
type NodeStatus struct {
	Capacity struct {
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
	} `json:"capacity"`
}

type Node struct {
	Metadata NodeMetadata `json:"metadata"`
	Status   NodeStatus   `json:"status"`
}

func getNodeExtra(resource Resource) NodeExtra {
	var e NodeExtra
	if resource.Extra != nil {
		e = resource.Extra.Copy().(NodeExtra)
	} else {
		cores, mem, creationTime := getCapacity(resource)
		e.CPUCapacity = cores * 1000
		e.MemCapacity = mem
		e.NodeCreationTimestamp = creationTime
		e.Uptime = time.Since(creationTime).Truncate(time.Second)
	}

	return e
}

func getCapacity(resource Resource) (int64, int64, time.Time) {
	var node Node
	err := yaml.Unmarshal([]byte(resource.RawJSON), &node)
	if err != nil {
		log.Fatalf("error parsing YAML: %v", err)
	}
	// Convert CPU to int
	cpuCores, err := strconv.Atoi(node.Status.Capacity.CPU)
	if err != nil {
		log.Fatalf("error parsing CPU: %v", err)
	}
	// Convert Memory to bytes
	memoryBytes, err := misc.ParseMemory(node.Status.Capacity.Memory)
	if err != nil {
		log.Fatalf("error parsing memory: %v : %s", err, node.Status.Capacity.Memory)
	}

	creationTime, err := time.Parse(time.RFC3339, node.Metadata.CreationTimestamp)
	if err != nil {
		creationTime = time.Now()
	}

	return int64(cpuCores), memoryBytes, creationTime
}

func getMetricsForNode(resource Resource) map[string]string {
	e := getNodeExtra(resource)

	metricsExtra := map[string]string{}
	lastNodeMetrics := lastNodeMetrics.Load()
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

func NodeTicker(podWatcher *PodWatcher, dao DAO, metricsClient *metrics.Clientset) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	m, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return
	}
	lastNodeMetrics.Store(m)

	// Get the current resources
	resources := dao.GetResourcesAt(time.Now(), "Node", "")
	for _, resource := range resources {
		updateResourceMetrics(podWatcher, dao, resource)
	}
}
