package main

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	result = append(result, RenderMapOfStrings("Labels:", service.Labels)...)
	result = append(result, RenderMapOfStrings("Annotations:", service.Annotations)...)

	// Selector
	if service.Spec.Selector != nil {
		result = append(result, RenderMapOfStrings("Selector:", service.Spec.Selector)...)
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

func (r ServiceRenderer) Render(resource Resource, details bool) []string {
	if details {
		return formatServiceDetails(resource.Object.(*corev1.Service))
	}

	return []string{resource.Key()}
}

type ServiceWatchMe struct {
}

func (n ServiceWatchMe) Tick() {
}

func (n ServiceWatchMe) Kind() string {
	return "Service"
}

func (n *ServiceWatchMe) Renderer() ResourceRenderer {
	return nil
}

func (n ServiceWatchMe) convert(obj runtime.Object) *corev1.Service {
	ret, ok := obj.(*corev1.Service)
	if !ok {
		return nil
	}
	return ret
}

func (n ServiceWatchMe) Valid(obj runtime.Object) bool {
	return n.convert(obj) != nil
}

func (n ServiceWatchMe) Add(obj runtime.Object) Resource {
	service := n.convert(obj)
	return NewResource(string(service.ObjectMeta.GetUID()), service.ObjectMeta.CreationTimestamp.Time, n.Kind(), service.Namespace, service.Name, service, ServiceRenderer{})

}
func (n ServiceWatchMe) Modified(obj runtime.Object) Resource {
	service := n.convert(obj)
	return NewResource(string(service.ObjectMeta.GetUID()), time.Now(), n.Kind(), service.Namespace, service.Name, service, ServiceRenderer{})

}
func (n ServiceWatchMe) Del(obj runtime.Object) Resource {
	service := n.convert(obj)
	return NewResource(string(service.ObjectMeta.GetUID()), time.Now(), n.Kind(), service.Namespace, service.Name, service, ServiceRenderer{})

}

func watchForService(watcher *K8sWatcher, k KhronosConn) {
	fmt.Println("Watching service . . .")
	watchChan, err := k.client.CoreV1().Services("").Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		panic(err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), ServiceWatchMe{})
}
