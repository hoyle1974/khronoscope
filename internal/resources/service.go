package resources

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/serializable"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ServiceExtra holds the complete service state in a serializable form for GOB encoding
type ServiceExtra struct {
	Type                  string
	ClusterIP             string
	ExternalIPs           []string
	Ports                 []string
	TargetPorts           []string
	Selector              []string
	Labels                []string
	Annotations           []string
	IPFamilyPolicy        string
	IPFamilies            []string
	IPs                   []string
	SessionAffinity       string
	InternalTrafficPolicy string
	ExternalTrafficPolicy string
	HealthCheckNodePort   int32
	LoadBalancerIP        string
	LoadBalancerClass     string
	Endpoints             []string
	Events                []string
	Age                   serializable.Time
}

// Copy performs a deep copy of ServiceExtra
func (p ServiceExtra) Copy() Copyable {
	return ServiceExtra{
		Type:                  p.Type,
		ClusterIP:             p.ClusterIP,
		ExternalIPs:           misc.DeepCopyArray(p.ExternalIPs),
		Ports:                 misc.DeepCopyArray(p.Ports),
		TargetPorts:           misc.DeepCopyArray(p.TargetPorts),
		Selector:              misc.DeepCopyArray(p.Selector),
		Labels:                misc.DeepCopyArray(p.Labels),
		Annotations:           misc.DeepCopyArray(p.Annotations),
		IPFamilyPolicy:        p.IPFamilyPolicy,
		IPFamilies:            misc.DeepCopyArray(p.IPFamilies),
		IPs:                   misc.DeepCopyArray(p.IPs),
		SessionAffinity:       p.SessionAffinity,
		InternalTrafficPolicy: p.InternalTrafficPolicy,
		ExternalTrafficPolicy: p.ExternalTrafficPolicy,
		HealthCheckNodePort:   p.HealthCheckNodePort,
		LoadBalancerIP:        p.LoadBalancerIP,
		LoadBalancerClass:     p.LoadBalancerClass,
		Endpoints:             misc.DeepCopyArray(p.Endpoints),
		Events:                misc.DeepCopyArray(p.Events),
		Age:                   p.Age,
	}
}

// newServiceExtra constructs a ServiceExtra from a Kubernetes Service
func newServiceExtra(svc *corev1.Service) ServiceExtra {
	if svc == nil {
		return ServiceExtra{}
	}

	// Process ports with named ports and protocols
	ports := make([]string, 0, len(svc.Spec.Ports))
	targetPorts := make([]string, 0, len(svc.Spec.Ports))
	for _, p := range svc.Spec.Ports {
		portStr := fmt.Sprintf("%s  %d/%s", p.Name, p.Port, p.Protocol)
		ports = append(ports, strings.TrimLeft(portStr, " "))

		targetPort := ""
		if p.TargetPort.StrVal != "" {
			targetPort = p.TargetPort.StrVal
		} else {
			targetPort = fmt.Sprintf("%d", p.TargetPort.IntVal)
		}
		targetPorts = append(targetPorts, fmt.Sprintf("%s/%s", targetPort, p.Protocol))
	}
	sort.Strings(ports)
	sort.Strings(targetPorts)

	// Process selector with proper formatting
	selector := make([]string, 0, len(svc.Spec.Selector))
	for k, v := range svc.Spec.Selector {
		selector = append(selector, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(selector)

	// Handle IP Families
	ipFamilies := make([]string, 0, len(svc.Spec.IPFamilies))
	for _, family := range svc.Spec.IPFamilies {
		ipFamilies = append(ipFamilies, string(family))
	}

	// Handle IP Addresses
	ips := make([]string, 0)
	if svc.Spec.ClusterIP != "" {
		ips = append(ips, svc.Spec.ClusterIP)
	}
	if len(svc.Spec.ClusterIPs) > 0 {
		ips = append(ips, svc.Spec.ClusterIPs...)
	}

	// Get IPFamilyPolicy as string
	ipFamilyPolicy := "<none>"
	if svc.Spec.IPFamilyPolicy != nil {
		ipFamilyPolicy = string(*svc.Spec.IPFamilyPolicy)
	}

	// Get InternalTrafficPolicy as string
	internalTrafficPolicy := "<none>"
	if svc.Spec.InternalTrafficPolicy != nil {
		internalTrafficPolicy = string(*svc.Spec.InternalTrafficPolicy)
	}

	return ServiceExtra{
		Type:                  string(svc.Spec.Type),
		ClusterIP:             svc.Spec.ClusterIP,
		ExternalIPs:           misc.DeepCopyArray(svc.Spec.ExternalIPs),
		Ports:                 ports,
		TargetPorts:           targetPorts,
		Selector:              selector,
		Labels:                misc.RenderMapOfStrings(svc.Labels),
		Annotations:           misc.RenderMapOfStrings(svc.Annotations),
		IPFamilyPolicy:        ipFamilyPolicy,
		IPFamilies:            ipFamilies,
		IPs:                   ips,
		SessionAffinity:       string(svc.Spec.SessionAffinity),
		InternalTrafficPolicy: internalTrafficPolicy,
		ExternalTrafficPolicy: string(svc.Spec.ExternalTrafficPolicy),
		HealthCheckNodePort:   svc.Spec.HealthCheckNodePort,
		LoadBalancerIP:        svc.Spec.LoadBalancerIP,
		LoadBalancerClass:     misc.FormatNilString(svc.Spec.LoadBalancerClass),
		Age:                   serializable.NewTime(svc.CreationTimestamp.Time),
	}
}

type ServiceRenderer struct{}

func renderServiceExtra(extra ServiceExtra) []string {
	output := []string{
		fmt.Sprintf("Type:                     %s", extra.Type),
		fmt.Sprintf("IP Family Policy:         %s", extra.IPFamilyPolicy),
		fmt.Sprintf("IP Families:              %s", misc.FormatNilArray(extra.IPFamilies)),
		fmt.Sprintf("IP:                       %s", extra.ClusterIP),
		fmt.Sprintf("IPs:                      %s", misc.FormatNilArray(extra.IPs)),
		fmt.Sprintf("External IPs:             %s", misc.FormatNilArray(extra.ExternalIPs)),
		fmt.Sprintf("Port:                     %s", misc.FormatNilArray(extra.Ports)),
		fmt.Sprintf("TargetPort:               %s", misc.FormatNilArray(extra.TargetPorts)),
		fmt.Sprintf("Selector:                 %s", misc.FormatNilArray(extra.Selector)),
		fmt.Sprintf("Session Affinity:         %s", extra.SessionAffinity),
		fmt.Sprintf("Internal Traffic Policy:  %s", extra.InternalTrafficPolicy),
		fmt.Sprintf("External Traffic Policy:  %s", extra.ExternalTrafficPolicy),
	}

	if extra.HealthCheckNodePort > 0 {
		output = append(output, fmt.Sprintf("Health Check Node Port:  %d", extra.HealthCheckNodePort))
	}

	if extra.LoadBalancerIP != "" {
		output = append(output, fmt.Sprintf("LoadBalancer IP:         %s", extra.LoadBalancerIP))
	}

	if extra.LoadBalancerClass != "" {
		output = append(output, fmt.Sprintf("LoadBalancer Class:      %s", extra.LoadBalancerClass))
	}

	// Add Labels section if present
	if len(extra.Labels) > 0 {
		output = append(output, "Labels:")
		for _, label := range extra.Labels {
			output = append(output, fmt.Sprintf("                          %s", label))
		}
	}

	// Add Annotations section if present
	if len(extra.Annotations) > 0 {
		output = append(output, "Annotations:")
		for _, annotation := range extra.Annotations {
			output = append(output, fmt.Sprintf("                          %s", annotation))
		}
	}

	return output
}

func (r ServiceRenderer) Render(resource Resource, details bool) []string {
	extra, ok := resource.Extra.(ServiceExtra)
	if !ok {
		return []string{"Error: Invalid extra type"}
	}

	if details {
		return renderServiceExtra(extra)
	}
	return []string{resource.Key()}
}

type ServiceWatcher struct{}

func (n ServiceWatcher) Tick()                      {}
func (n ServiceWatcher) Kind() string               { return "Service" }
func (n ServiceWatcher) Renderer() ResourceRenderer { return ServiceRenderer{} }

func (n ServiceWatcher) convert(obj runtime.Object) *corev1.Service {
	if obj == nil {
		return nil
	}
	ret, ok := obj.(*corev1.Service)
	if !ok {
		return nil
	}
	return ret
}

func (n ServiceWatcher) ToResource(obj runtime.Object) Resource {
	svc := n.convert(obj)
	if svc == nil {
		return Resource{}
	}
	extra := newServiceExtra(svc)
	return NewK8sResource(n.Kind(), svc, extra)
}

func watchForService(watcher *K8sWatcher, k conn.KhronosConn, ns string) error {
	watchChan, err := k.Client.CoreV1().Services(ns).Watch(context.Background(), v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to watch services: %w", err)
	}

	go watcher.registerEventWatcher(watchChan.ResultChan(), ServiceWatcher{})
	return nil
}
