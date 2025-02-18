package resources

import (
	"context"
	"fmt"
	"sort"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/misc/format"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
)

// ServiceExtra holds all necessary service state
type ServiceExtra struct {
	Name              string
	Namespace         string
	UID               string
	CreationTimestamp string
	Type              string
	ClusterIP         string
	ExternalIPs       []string
	Ports             []string
	Selector          []string
	Labels            []string
	Annotations       []string
}

// Copy creates a deep copy of ServiceExtra
func (p ServiceExtra) Copy() Copyable {
	return ServiceExtra{
		Name:              p.Name,
		Namespace:         p.Namespace,
		UID:               p.UID,
		CreationTimestamp: p.CreationTimestamp,
		Type:              p.Type,
		ClusterIP:         p.ClusterIP,
		ExternalIPs:       misc.DeepCopyArray(p.ExternalIPs),
		Ports:             misc.DeepCopyArray(p.Ports),
		Selector:          misc.DeepCopyArray(p.Selector),
		Labels:            misc.DeepCopyArray(p.Labels),
		Annotations:       misc.DeepCopyArray(p.Annotations),
	}
}

// newServiceExtra constructs a ServiceExtra from a *corev1.Service
func newServiceExtra(svc *corev1.Service) ServiceExtra {
	ports := make([]string, len(svc.Spec.Ports))
	for i, p := range svc.Spec.Ports {
		ports[i] = fmt.Sprintf("%d/%s", p.Port, p.Protocol)
	}
	sort.Strings(ports)

	selector := make([]string, 0, len(svc.Spec.Selector))
	for k, v := range svc.Spec.Selector {
		selector = append(selector, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(selector)

	return ServiceExtra{
		Name:              svc.Name,
		Namespace:         svc.Namespace,
		UID:               string(svc.UID),
		CreationTimestamp: duration.HumanDuration(v1.Now().Sub(svc.CreationTimestamp.Time)),
		Type:              string(svc.Spec.Type),
		ClusterIP:         svc.Spec.ClusterIP,
		ExternalIPs:       misc.DeepCopyArray(svc.Spec.ExternalIPs),
		Ports:             ports,
		Selector:          selector,
		Labels:            misc.RenderMapOfStrings("Labels", svc.Labels),
		Annotations:       misc.RenderMapOfStrings("Annotations", svc.Annotations),
	}
}

type ServiceRenderer struct {
}

// renderServiceExtra formats the data similar to `kubectl get services`
func renderServiceExtra(extra ServiceExtra) []string {
	output := []string{
		fmt.Sprintf("Name:          %s", extra.Name),
		fmt.Sprintf("Namespace:     %s", extra.Namespace),
		fmt.Sprintf("UID:           %s", extra.UID),
		fmt.Sprintf("Created:       %s ago", extra.CreationTimestamp),
		fmt.Sprintf("Type:          %s", extra.Type),
		fmt.Sprintf("ClusterIP:     %s", extra.ClusterIP),
		fmt.Sprintf("ExternalIPs:   %s", misc.FormatArray(extra.ExternalIPs)),
		fmt.Sprintf("Ports:         %s", misc.FormatArray(extra.Ports)),
		fmt.Sprintf("Selector:      %s", misc.FormatArray(extra.Selector)),
	}

	output = append(output, extra.Labels...)
	output = append(output, extra.Annotations...)

	return output
}

func (r ServiceRenderer) Render(resource Resource, details bool) []string {
	extra := resource.Extra.(ServiceExtra)

	if details {
		return renderServiceExtra(extra)
	}

	return []string{resource.Key()}
}

type ServiceWatcher struct {
}

func (n ServiceWatcher) Tick() {
}

func (n ServiceWatcher) Kind() string {
	return "Service"
}

func (n ServiceWatcher) Renderer() ResourceRenderer {
	return ServiceRenderer{}
}

func (n ServiceWatcher) convert(obj runtime.Object) *corev1.Service {
	ret, ok := obj.(*corev1.Service)
	if !ok {
		return nil
	}
	return ret
}

func (n ServiceWatcher) ToResource(obj runtime.Object) Resource {
	s := n.convert(obj)
	extra := newServiceExtra(s)
	return NewK8sResource(n.Kind(), s, format.FormatServiceDetails(n.convert(obj)), extra)
}

func watchForService(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	watchChan, err := k.Client.CoreV1().Services(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), ServiceWatcher{})

	return nil
}
