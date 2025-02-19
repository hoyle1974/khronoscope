package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DaemonSetExtra struct {
	Name            string
	Namespace       string
	Labels          []string
	Annotations     []string
	Selector        []string
	NodeSelector    []string
	Tolerations     []string
	DesiredNumber   int32
	CurrentNumber   int32
	ReadyNumber     int32
	UpdatedNumber   int32
	AvailableNumber int32
	Age             string
	Containers      []string
	Images          []string
}

func newDaemonSetExtra(ds *appsv1.DaemonSet) DaemonSetExtra {
	labels := misc.RenderMapOfStrings(ds.Labels)
	annotations := misc.RenderMapOfStrings(ds.Annotations)
	selector := misc.RenderMapOfStrings(ds.Spec.Selector.MatchLabels)
	nodeSelector := misc.RenderMapOfStrings(ds.Spec.Template.Spec.NodeSelector)
	tolerations := []string{}
	for _, t := range ds.Spec.Template.Spec.Tolerations {
		tolerations = append(tolerations, fmt.Sprintf("%s=%s:%s", t.Key, t.Value, t.Operator))
	}

	containers := []string{}
	images := []string{}
	for _, c := range ds.Spec.Template.Spec.Containers {
		containers = append(containers, c.Name)
		images = append(images, c.Image)
	}

	age := metav1.Now().Sub(ds.CreationTimestamp.Time).String()

	return DaemonSetExtra{
		Name:            ds.Name,
		Namespace:       ds.Namespace,
		Labels:          labels,
		Annotations:     annotations,
		Selector:        selector,
		NodeSelector:    nodeSelector,
		Tolerations:     tolerations,
		DesiredNumber:   ds.Status.DesiredNumberScheduled,
		CurrentNumber:   ds.Status.CurrentNumberScheduled,
		ReadyNumber:     ds.Status.NumberReady,
		UpdatedNumber:   ds.Status.UpdatedNumberScheduled,
		AvailableNumber: ds.Status.NumberAvailable,
		Age:             age,
		Containers:      containers,
		Images:          images,
	}
}

func (p DaemonSetExtra) Copy() Copyable {
	return DaemonSetExtra{
		Name:            p.Name,
		Namespace:       p.Namespace,
		Labels:          misc.DeepCopyArray(p.Labels),
		Annotations:     misc.DeepCopyArray(p.Annotations),
		Selector:        misc.DeepCopyArray(p.Selector),
		NodeSelector:    misc.DeepCopyArray(p.NodeSelector),
		Tolerations:     misc.DeepCopyArray(p.Tolerations),
		DesiredNumber:   p.DesiredNumber,
		CurrentNumber:   p.CurrentNumber,
		ReadyNumber:     p.ReadyNumber,
		UpdatedNumber:   p.UpdatedNumber,
		AvailableNumber: p.AvailableNumber,
		Age:             p.Age,
		Containers:      misc.DeepCopyArray(p.Containers),
		Images:          misc.DeepCopyArray(p.Images),
	}
}

func renderDeamonSet(extra DaemonSetExtra) []string {
	lines := []string{}
	lines = append(lines, "NAME\tNAMESPACE\tDESIRED\tCURRENT\tREADY\tUP-TO-DATE\tAVAILABLE\tAGE")
	lines = append(lines, fmt.Sprintf("%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s",
		extra.Name, extra.Namespace, extra.DesiredNumber, extra.CurrentNumber, extra.ReadyNumber, extra.UpdatedNumber, extra.AvailableNumber, extra.Age))

	if len(extra.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("Labels: %s", strings.Join(extra.Labels, ", ")))
	}
	if len(extra.Annotations) > 0 {
		lines = append(lines, fmt.Sprintf("Annotations: %s", strings.Join(extra.Annotations, ", ")))
	}
	if len(extra.Selector) > 0 {
		lines = append(lines, fmt.Sprintf("Selector: %s", strings.Join(extra.Selector, ", ")))
	}
	if len(extra.NodeSelector) > 0 {
		lines = append(lines, fmt.Sprintf("NodeSelector: %s", strings.Join(extra.NodeSelector, ", ")))
	}
	if len(extra.Tolerations) > 0 {
		lines = append(lines, fmt.Sprintf("Tolerations: %s", strings.Join(extra.Tolerations, ", ")))
	}
	if len(extra.Containers) > 0 {
		lines = append(lines, fmt.Sprintf("Containers: %s", strings.Join(extra.Containers, ", ")))
	}
	if len(extra.Images) > 0 {
		lines = append(lines, fmt.Sprintf("Images: %s", strings.Join(extra.Images, ", ")))
	}

	return lines
}

type DaemonSetRenderer struct {
}

func (r DaemonSetRenderer) Render(resource Resource, details bool) []string {
	extra := resource.Extra.(DaemonSetExtra)

	if details {
		return renderDeamonSet(extra)
	}

	return []string{resource.Key()}
}

type DaemonSetWatcher struct {
}

func (n DaemonSetWatcher) Tick() {
}

func (n DaemonSetWatcher) Kind() string {
	return "DaemonSet"
}

func (n DaemonSetWatcher) Renderer() ResourceRenderer {
	return DaemonSetRenderer{}
}

func (n DaemonSetWatcher) convert(obj runtime.Object) *appsv1.DaemonSet {
	ret, ok := obj.(*appsv1.DaemonSet)
	if !ok {
		return nil
	}
	return ret
}

func (n DaemonSetWatcher) ToResource(obj runtime.Object) Resource {
	d := n.convert(obj)

	extra := newDaemonSetExtra(d)
	return NewK8sResource(n.Kind(), d, extra)
}

func watchForDaemonSet(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	watchChan, err := k.Client.AppsV1().DaemonSets(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), DaemonSetWatcher{})

	return nil
}
