package resources

import (
	"context"
	"fmt"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
)

// ConfigMapExtra holds the complete config map state in a serializable form for GOB encoding
type ConfigMapExtra struct {
	Name        string
	Namespace   string
	Labels      []string
	Annotations []string
	Data        map[string]string
	BinaryData  map[string]string
	Immutable   string
	CreateTime  serializable.Time
}

// Copy performs a deep copy of ConfigMapExtra
func (p ConfigMapExtra) Copy() Copyable {
	return ConfigMapExtra{
		Name:        p.Name,
		Namespace:   p.Namespace,
		Labels:      misc.DeepCopyArray(p.Labels),
		Annotations: misc.DeepCopyArray(p.Annotations),
		Data:        misc.DeepCopyMap(p.Data),
		BinaryData:  misc.DeepCopyMap(p.BinaryData),
		Immutable:   p.Immutable,
		CreateTime:  p.CreateTime,
	}
}

// newConfigMapExtra constructs a ConfigMapExtra from a Kubernetes ConfigMap
func newConfigMapExtra(cm *corev1.ConfigMap) ConfigMapExtra {
	if cm == nil {
		return ConfigMapExtra{}
	}

	data := make(map[string]string, len(cm.Data))
	for k, v := range cm.Data {
		data[k] = string(v)
	}

	binaryData := make(map[string]string, len(cm.BinaryData))
	for k, v := range cm.BinaryData {
		binaryData[k] = string(v)
	}

	immutable := "<empty>"
	if cm.Immutable != nil {
		immutable = fmt.Sprintf("%v", *cm.Immutable)
	}

	return ConfigMapExtra{
		Name:        cm.Name,
		Namespace:   cm.Namespace,
		Labels:      misc.RenderMapOfStrings(cm.Labels),
		Annotations: misc.RenderMapOfStrings(cm.Annotations),
		Data:        data,
		BinaryData:  binaryData,
		Immutable:   immutable,
		CreateTime:  serializable.NewTime(cm.CreationTimestamp.Time),
	}
}

type ConfigMapRenderer struct{}

func renderConfigMapExtra(extra ConfigMapExtra) []string {

	createTime := duration.HumanDuration(v1.Now().Sub(extra.CreateTime.Time))

	output := []string{
		fmt.Sprintf("Name:                     %s", extra.Name),
		fmt.Sprintf("Namespace:                %s", extra.Namespace),
		fmt.Sprintf("Labels:                   %s", misc.FormatNilArray(extra.Labels)),
		fmt.Sprintf("Annotations:              %s", misc.FormatNilArray(extra.Annotations)),
		fmt.Sprintf("Immutable:                %s", extra.Immutable),
		fmt.Sprintf("Creation Time:            %s", createTime),
	}

	// Add Data section if present
	if len(extra.Data) > 0 {
		output = append(output, "Data:")
		for k, v := range extra.Data {
			output = append(output, fmt.Sprintf("  %s: %s", k, v))
		}
	}

	// Add BinaryData section if present
	if len(extra.BinaryData) > 0 {
		output = append(output, "Binary Data:")
		for k, v := range extra.BinaryData {
			output = append(output, fmt.Sprintf("  %s: %s", k, v))
		}
	}

	return output
}

func (r ConfigMapRenderer) Render(resource Resource, details bool) []string {
	extra, ok := resource.Extra.(ConfigMapExtra)
	if !ok {
		return []string{"Error: Invalid extra type"}
	}

	if details {
		return renderConfigMapExtra(extra)
	}
	return []string{resource.Key()}
}

type ConfigMapWatcher struct{}

func (n ConfigMapWatcher) Tick()                      {}
func (n ConfigMapWatcher) Kind() string               { return "ConfigMap" }
func (n ConfigMapWatcher) Renderer() ResourceRenderer { return ConfigMapRenderer{} }

func (n ConfigMapWatcher) convert(obj runtime.Object) *corev1.ConfigMap {
	if obj == nil {
		return nil
	}
	ret, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil
	}
	return ret
}

func (n ConfigMapWatcher) ToResource(obj runtime.Object) Resource {
	cm := n.convert(obj)
	if cm == nil {
		return Resource{}
	}
	extra := newConfigMapExtra(cm)
	return NewK8sResource(n.Kind(), cm, extra)
}

func watchForConfigMap(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	watchChan, err := k.Client.CoreV1().ConfigMaps(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to watch config maps: %w", err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), ConfigMapWatcher{})
	return nil
}
