package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hoyle1974/khronoscope/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ServiceRenderer struct {
}

func formatServiceDetails(service *corev1.Service) []string {
	var result []string

	// Basic details
	result = append(result, fmt.Sprintf("Name:           %s", service.Name))
	result = append(result, fmt.Sprintf("Namespace:      %s", service.Namespace))
	result = append(result, misc.RenderMapOfStrings("Labels:", service.Labels)...)
	result = append(result, misc.RenderMapOfStrings("Annotations:", service.Annotations)...)

	// Selector
	if service.Spec.Selector != nil {
		result = append(result, misc.RenderMapOfStrings("Selector:", service.Spec.Selector)...)
	}

	// Type
	result = append(result, fmt.Sprintf("Type:           %s", service.Spec.Type))

	// Cluster IP
	result = append(result, fmt.Sprintf("Cluster IP:     %s", service.Spec.ClusterIP))

	// Ports
	if len(service.Spec.Ports) > 0 {
		result = append(result, "Ports:")
		for _, port := range service.Spec.Ports {
			result = append(result, fmt.Sprintf("  Port:        %d", port.Port))
			result = append(result, fmt.Sprintf("  Target Port: %d", port.TargetPort.IntValue()))
			result = append(result, fmt.Sprintf("  Protocol:    %s", port.Protocol))
		}
	}

	// External IPs
	if len(service.Spec.ExternalIPs) > 0 {
		result = append(result, fmt.Sprintf("External IPs:   %s", strings.Join(service.Spec.ExternalIPs, ", ")))
	}

	// Session Affinity
	result = append(result, fmt.Sprintf("Session Affinity: %s", service.Spec.SessionAffinity))

	// Events
	result = append(result, "Events:         <none>")

	return result
}

func (r ServiceRenderer) Render(resource Resource, obj any, details bool) []string {
	if details {
		return formatServiceDetails(obj.(*corev1.Service))
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
	return nil
}

func (n ServiceWatcher) convert(obj runtime.Object) *corev1.Service {
	ret, ok := obj.(*corev1.Service)
	if !ok {
		return nil
	}
	return ret
}

func (n ServiceWatcher) ToResource(obj runtime.Object) Resource {
	return NewK8sResource(n.Kind(), n.convert(obj), formatServiceDetails(n.convert(obj)), nil)
}

func watchForService(watcher *K8sWatcher, k conn.KhronosConn) {
	watchChan, err := k.Client.CoreV1().Services("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), ServiceWatcher{})
}
