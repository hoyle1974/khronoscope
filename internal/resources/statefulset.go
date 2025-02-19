package resources

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type StatefulSetExtra struct {
	Name              string
	Namespace         string
	UID               string
	CreationTimestamp time.Time
	Labels            []string
	Annotations       []string
}

func (p StatefulSetExtra) Copy() Copyable {
	return StatefulSetExtra{
		Name:              p.Name,
		Namespace:         p.Namespace,
		UID:               p.UID,
		CreationTimestamp: p.CreationTimestamp,
		Labels:            misc.DeepCopyArray(p.Labels),
		Annotations:       misc.DeepCopyArray(p.Annotations),
	}
}

func newStatefulSetExtra(ss *appsv1.StatefulSet) StatefulSetExtra {
	labels := misc.RenderMapOfStrings(ss.Labels)
	annotations := misc.RenderMapOfStrings(ss.Annotations)

	return StatefulSetExtra{
		Name:              ss.Name,
		Namespace:         ss.Namespace,
		UID:               string(ss.UID),
		CreationTimestamp: ss.CreationTimestamp.Time,
		Labels:            labels,
		Annotations:       annotations,
	}
}

func renderStatefulSetExtra(extra StatefulSetExtra) []string {
	output := []string{
		fmt.Sprintf("Name: %s", extra.Name),
		fmt.Sprintf("Namespace: %s", extra.Namespace),
		fmt.Sprintf("UID: %s", extra.UID),
		fmt.Sprintf("CreationTimestamp: %s", extra.CreationTimestamp.Format(time.RFC3339)),
	}
	output = append(output, extra.Labels...)
	output = append(output, extra.Annotations...)
	return output
}

type StatefulSetRenderer struct {
}

func (r StatefulSetRenderer) Render(resource Resource, details bool) []string {
	extra := resource.Extra.(StatefulSetExtra)

	if details {
		return renderStatefulSetExtra(extra)
	}

	return []string{resource.Key()}
}

type StatefulSetWatcher struct {
}

func (n StatefulSetWatcher) Tick() {
}

func (n StatefulSetWatcher) Kind() string {
	return "StatefulSet"
}

func (n StatefulSetWatcher) Renderer() ResourceRenderer {
	return StatefulSetRenderer{}
}

func (n StatefulSetWatcher) convert(obj runtime.Object) *appsv1.StatefulSet {
	ret, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		return nil
	}
	return ret
}

func (n StatefulSetWatcher) ToResource(obj runtime.Object) Resource {
	ss := n.convert(obj)
	extra := newStatefulSetExtra(ss)
	return NewK8sResource(n.Kind(), ss, extra)
}

// watchForStatefulSet now also fetches pods associated with the StatefulSet
func watchForStatefulSet(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	// Watch for StatefulSet events
	watchChan, err := k.Client.AppsV1().StatefulSets(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	// Fetch pods related to the StatefulSet by label selector
	go func() {
		for event := range watchChan.ResultChan() {
			if event.Type == "ADDED" || event.Type == "MODIFIED" {
				// Fetch the associated pods
				ss, ok := event.Object.(*appsv1.StatefulSet)
				if !ok {
					continue
				}

				labelSelector := metav1.FormatLabelSelector(ss.Spec.Selector)
				pods, err := k.Client.CoreV1().Pods(ns).List(context.Background(), v1.ListOptions{LabelSelector: labelSelector})
				if err != nil {
					continue
				}

				podNames := make([]string, len(pods.Items))
				for i, pod := range pods.Items {
					podNames[i] = pod.Name
				}

				sort.Strings(podNames)
				// Handle the pod names, you can store them or handle them in the required way
				fmt.Printf("Pod names for StatefulSet %s: %v\n", ss.Name, podNames)
			}
		}
	}()

	go watcher.registerEventWatcher(watchChan.ResultChan(), StatefulSetWatcher{})

	return nil
}
