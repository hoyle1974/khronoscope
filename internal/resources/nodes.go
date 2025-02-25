package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"

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

// mapToPodExtra converts a map[string]any to a PodExtra struct.
func nodeExtraFromMap(e map[string]any) (NodeExtra, bool) {
	if e == nil {
		return NodeExtra{}, false
	}

	// Convert the map to JSON bytes
	jsonBytes, err := json.Marshal(e)
	if err != nil {
		panic(err)
		// return NodeExtra{}, false
	}

	// Unmarshal the JSON bytes into a PodExtra struct
	var ne NodeExtra
	err = json.Unmarshal(jsonBytes, &ne)
	if err != nil {
		panic(err)
		// return NodeExtra{}, false
	}

	return ne, true
}

// podExtraToMap converts a PodExtra struct back to a map[string]any.
func (ne NodeExtra) toMap() (map[string]any, bool) {
	// Convert the PodExtra struct to JSON bytes
	jsonBytes, err := json.Marshal(ne)
	if err != nil {
		panic(err)
		// return nil, false
	}

	// Unmarshal the JSON bytes into a map[string]any
	var m map[string]any
	err = json.Unmarshal(jsonBytes, &m)
	if err != nil {
		panic(err)
		// return nil, false
	}

	return m, true
}

type nodeRenderer struct {
	dao DAO
}

func (r nodeRenderer) Render(resource Resource, details bool) []string {
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

		s, _ := misc.PrettyPrintYamlFromAny(resource.RawJSON)
		ret = append(ret, strings.Split(s, "\n")...)

		return ret
	}

	out := fmt.Sprintf("%v", extra.NodeMetrics[resource.Name])
	out += " " + resource.Name
	out += fmt.Sprintf(" %v", extra.Uptime)

	return []string{out}

}

func updateNodeResourceMetrics(dao DAO, resource Resource) {
	extra := getNodeExtra(resource)

	resource.Timestamp = serializable.Time{Time: time.Now()}

	metricsExtra := getMetricsForNode(resource)
	if len(metricsExtra) > 0 {
		extra.NodeMetrics = metricsExtra
		extra.Uptime = time.Since(extra.NodeCreationTimestamp).Truncate(time.Second)
	}

	// Find pods on node
	resources := dao.GetResourcesAt(resource.Timestamp.Time, "Pod", resource.Namespace)
	for _, podResource := range resources {
		if getPodExtra(podResource).NodeName == resource.Name {
			podMetrics := getPodMetricsForPod(podResource)
			extra.PodMetrics[podResource.Namespace+"/"+podResource.Name] = podMetrics
		}
	}

	m, _ := extra.toMap()
	resource.Extra = m

	dao.UpdateResource(resource)

}

var lastNodeMetrics atomic.Pointer[v1beta1.NodeMetricsList]

type nodeMetadata struct {
	CreationTimestamp string `json:"creationTimestamp"`
}
type nodeStatus struct {
	Capacity struct {
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
	} `json:"capacity"`
}

type node struct {
	Metadata nodeMetadata `json:"metadata"`
	Status   nodeStatus   `json:"status"`
}

func getNodeExtra(resource Resource) NodeExtra {
	var extra NodeExtra
	if resource.Extra != nil {
		extra, _ = nodeExtraFromMap(resource.Extra)
	} else {
		cores, mem, creationTime := getNodeCapacity(resource)
		extra.CPUCapacity = cores * 1000
		extra.MemCapacity = mem
		extra.NodeCreationTimestamp = creationTime
		extra.Uptime = time.Since(creationTime).Truncate(time.Second)
		extra.PodMetrics = map[string]map[string]PodMetric{}
	}

	return extra
}

func getNodeCapacity(resource Resource) (int64, int64, time.Time) {
	var node node

	// TODO make this better
	b, _ := json.Marshal(resource.RawJSON)

	err := yaml.Unmarshal(b, &node)
	if err != nil {
		log.Panic().Err(err).Msg("error parsing YAML")
	}
	// Convert CPU to int
	cpuCores, err := strconv.Atoi(node.Status.Capacity.CPU)
	if err != nil {
		log.Panic().Err(err).Msg("error parsing CPU")
	}
	// Convert Memory to bytes
	memoryBytes, err := misc.ParseMemory(node.Status.Capacity.Memory)
	if err != nil {
		log.Panic().Err(err).Any("Memory", node.Status.Capacity.Memory).Msg("error parsing memory")
	}

	creationTime, err := time.Parse(time.RFC3339, node.Metadata.CreationTimestamp)
	if err != nil {
		creationTime = time.Now()
	}

	return int64(cpuCores), memoryBytes, creationTime
}

func getMetricsForNode(resource Resource) map[string]string {
	extra := getNodeExtra(resource)

	metricsExtra := map[string]string{}
	lastNodeMetrics := lastNodeMetrics.Load()
	if lastNodeMetrics == nil {
		return metricsExtra
	}
	for _, nodeMetrics := range lastNodeMetrics.Items {
		if nodeMetrics.Name == resource.Name {
			cpuUsage := nodeMetrics.Usage[corev1.ResourceCPU]
			memUsage := nodeMetrics.Usage[corev1.ResourceMemory]

			cpuPercentage := calculatePercentage(cpuUsage.MilliValue(), extra.CPUCapacity)
			memPercentage := calculatePercentage(memUsage.Value(), extra.MemCapacity)

			metricsExtra[resource.Name] = fmt.Sprintf("%s %s", misc.RenderProgressBar("CPU", cpuPercentage), misc.RenderProgressBar("Mem", memPercentage))

			return metricsExtra
		}
	}
	return metricsExtra
}

func nodeTicker(dao DAO, metricsClient *metrics.Clientset) {
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
		updateNodeResourceMetrics(dao, resource)
	}
}
