package resources

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ConfigMapExtra struct {
	Name              string
	Namespace         string
	UID               string
	CreationTimestamp time.Time
	Labels            []string
	Annotations       []string
	DataKeys          []string
	BinaryDataKeys    []string
}

func (p ConfigMapExtra) Copy() Copyable {
	return ConfigMapExtra{
		Name:              p.Name,
		Namespace:         p.Namespace,
		UID:               p.UID,
		CreationTimestamp: p.CreationTimestamp,
		Labels:            misc.DeepCopyArray(p.Labels),
		Annotations:       misc.DeepCopyArray(p.Annotations),
		DataKeys:          misc.DeepCopyArray(p.DataKeys),
		BinaryDataKeys:    misc.DeepCopyArray(p.BinaryDataKeys),
	}
}

func newConfigMapExtra(cm *corev1.ConfigMap) ConfigMapExtra {
	labels := misc.RenderMapOfStrings("Labels", cm.Labels)
	annotations := misc.RenderMapOfStrings("Annotations", cm.Annotations)

	dataKeys := make([]string, 0, len(cm.Data))
	for k := range cm.Data {
		dataKeys = append(dataKeys, k)
	}
	binaryDataKeys := make([]string, 0, len(cm.BinaryData))
	for k := range cm.BinaryData {
		binaryDataKeys = append(binaryDataKeys, k)
	}

	sort.Strings(dataKeys)
	sort.Strings(binaryDataKeys)

	return ConfigMapExtra{
		Name:              cm.Name,
		Namespace:         cm.Namespace,
		UID:               string(cm.UID),
		CreationTimestamp: cm.CreationTimestamp.Time,
		Labels:            labels,
		Annotations:       annotations,
		DataKeys:          dataKeys,
		BinaryDataKeys:    binaryDataKeys,
	}
}

func renderConfigMapExtra(extra ConfigMapExtra) []string {
	output := []string{
		fmt.Sprintf("Name: %s", extra.Name),
		fmt.Sprintf("Namespace: %s", extra.Namespace),
		fmt.Sprintf("UID: %s", extra.UID),
		fmt.Sprintf("CreationTimestamp: %s", extra.CreationTimestamp.Format(time.RFC3339)),
	}
	output = append(output, extra.Labels...)
	output = append(output, extra.Annotations...)
	output = append(output, fmt.Sprintf("Data keys: %v", extra.DataKeys))
	output = append(output, fmt.Sprintf("BinaryData keys: %v", extra.BinaryDataKeys))
	return output
}

type ConfigMapRenderer struct {
}

func (r ConfigMapRenderer) Render(resource Resource, details bool) []string {
	extra := resource.Extra.(ConfigMapExtra)

	if details {
		return renderConfigMapExtra(extra)
	}

	return []string{resource.Key()}
}

type ConfigMapWatcher struct {
}

func (n ConfigMapWatcher) Tick() {
}

func (n ConfigMapWatcher) Kind() string {
	return "ConfigMap"
}

func (n ConfigMapWatcher) Renderer() ResourceRenderer {
	return ConfigMapRenderer{}
}

func (n ConfigMapWatcher) convert(obj runtime.Object) *corev1.ConfigMap {
	ret, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return nil
	}
	return ret
}

func (n ConfigMapWatcher) ToResource(obj runtime.Object) Resource {
	cm := n.convert(obj)
	extra := newConfigMapExtra(cm)
	return NewK8sResource(n.Kind(), cm, []string{}, extra)
}

func watchForConfigMap(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	watchChan, err := k.Client.CoreV1().ConfigMaps(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), ConfigMapWatcher{})

	return nil
}
