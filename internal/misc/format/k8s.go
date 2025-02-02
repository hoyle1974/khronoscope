package format

import (
	"fmt"
	"strings"

	"github.com/hoyle1974/khronoscope/internal/misc"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// FormatDaemonSetDetails formats the details of a DaemonSet for display
func FormatDaemonSetDetails(ds *appsv1.DaemonSet) []string {
	var details []string

	details = append(details, fmt.Sprintf("Desired: %d", ds.Status.DesiredNumberScheduled))
	details = append(details, fmt.Sprintf("Current: %d", ds.Status.CurrentNumberScheduled))
	details = append(details, fmt.Sprintf("Ready: %d", ds.Status.NumberReady))
	details = append(details, fmt.Sprintf("Up-to-date: %d", ds.Status.UpdatedNumberScheduled))
	details = append(details, fmt.Sprintf("Available: %d", ds.Status.NumberAvailable))

	return details
}

// FormatNamespaceDetails formats the details of a Namespace for display
func FormatNamespaceDetails(namespace *corev1.Namespace) []string {
	var details []string

	details = append(details, fmt.Sprintf("Status: %s", namespace.Status.Phase))
	for _, condition := range namespace.Status.Conditions {
		details = append(details, fmt.Sprintf("%s: %s", condition.Type, condition.Status))
	}

	return details
}

// FormatDeploymentDetails formats the details of a Deployment for display
func FormatDeploymentDetails(deployment *appsv1.Deployment) []string {
	var details []string

	details = append(details, fmt.Sprintf("Replicas: %d", *deployment.Spec.Replicas))
	details = append(details, fmt.Sprintf("Ready: %d", deployment.Status.ReadyReplicas))
	details = append(details, fmt.Sprintf("Up-to-date: %d", deployment.Status.UpdatedReplicas))
	details = append(details, fmt.Sprintf("Available: %d", deployment.Status.AvailableReplicas))

	if deployment.Status.UnavailableReplicas > 0 {
		details = append(details, fmt.Sprintf("Unavailable: %d", deployment.Status.UnavailableReplicas))
	}

	return details
}

// FormatReplicaSetDetails formats the details of a ReplicaSet for display
func FormatReplicaSetDetails(rs *appsv1.ReplicaSet) []string {
	var details []string

	details = append(details, fmt.Sprintf("Replicas: %d", *rs.Spec.Replicas))
	details = append(details, fmt.Sprintf("Ready: %d", rs.Status.ReadyReplicas))
	details = append(details, fmt.Sprintf("Available: %d", rs.Status.AvailableReplicas))

	if rs.Status.Conditions != nil {
		for _, condition := range rs.Status.Conditions {
			details = append(details, fmt.Sprintf("%s: %s", condition.Type, condition.Status))
		}
	}

	return details
}

// FormatServiceDetails formats the details of a Service for display
func FormatServiceDetails(service *corev1.Service) []string {
	var details []string

	details = append(details, fmt.Sprintf("Type: %s", service.Spec.Type))
	if service.Spec.ClusterIP != "" {
		details = append(details, fmt.Sprintf("ClusterIP: %s", service.Spec.ClusterIP))
	}

	for _, port := range service.Spec.Ports {
		portDetail := fmt.Sprintf("Port: %d", port.Port)
		if port.TargetPort.String() != "" {
			portDetail += fmt.Sprintf("->%s", port.TargetPort.String())
		}
		if port.NodePort != 0 {
			portDetail += fmt.Sprintf(" NodePort: %d", port.NodePort)
		}
		details = append(details, portDetail)
	}

	return details
}

// FormatNodeDetails formats the details of a Node for display
func FormatNodeDetails(node *corev1.Node) []string {
	out := []string{}

	// Get node roles
	roles := []string{}
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		roles = append(roles, "<none>")
	}
	out = append(out, fmt.Sprintf("Roles: %s", strings.Join(roles, ",")))

	out = append(out, misc.RenderMapOfStrings("Labels:", node.GetLabels())...)
	out = append(out, misc.RenderMapOfStrings("Annotations:", node.GetAnnotations())...)

	out = append(out, "\nSystem Info:")
	out = append(out, fmt.Sprintf("  Machine ID:               %s", node.Status.NodeInfo.MachineID))
	out = append(out, fmt.Sprintf("  System UUID:              %s", node.Status.NodeInfo.SystemUUID))
	out = append(out, fmt.Sprintf("  Boot ID:                  %s", node.Status.NodeInfo.BootID))
	out = append(out, fmt.Sprintf("  Kernel Version:           %s", node.Status.NodeInfo.KernelVersion))
	out = append(out, fmt.Sprintf("  OS Image:                 %s", node.Status.NodeInfo.OSImage))
	out = append(out, fmt.Sprintf("  Container Runtime Version: %s", node.Status.NodeInfo.ContainerRuntimeVersion))
	out = append(out, fmt.Sprintf("  Kubelet Version:          %s", node.Status.NodeInfo.KubeletVersion))
	// out = append(out, fmt.Sprintf("  Kube-Proxy Version:       %s", node.Status.NodeInfo.KubeProxyVersion))
	out = append(out, fmt.Sprintf("  Operating System:         %s", node.Status.NodeInfo.OperatingSystem))
	out = append(out, fmt.Sprintf("  Architecture:             %s", node.Status.NodeInfo.Architecture))

	out = append(out, "\nConditions:")
	for _, condition := range node.Status.Conditions {
		out = append(out, fmt.Sprintf("  %v", condition.Type))
		out = append(out, fmt.Sprintf("    Status: %v", condition.Status))
		out = append(out, fmt.Sprintf("    Reason: %v", condition.Reason))
		out = append(out, fmt.Sprintf("    Message: %v", condition.Message))
	}

	out = append(out, "\nAddresses:")
	for _, address := range node.Status.Addresses {
		out = append(out, fmt.Sprintf("  %s: %s", address.Type, address.Address))
	}

	out = append(out, "\nCapacity:")
	for resource, quantity := range node.Status.Capacity {
		out = append(out, fmt.Sprintf("  %s: %s", resource, quantity.String()))
	}

	out = append(out, "\nAllocatable:")
	for resource, quantity := range node.Status.Allocatable {
		out = append(out, fmt.Sprintf("  %s: %s", resource, quantity.String()))
	}

	return out
}

// FormatPodDetails formats the details of a Pod for display
func FormatPodDetails(pod *corev1.Pod) []string {
	out := []string{}

	out = append(out, fmt.Sprintf("Name:         %s", pod.Name))
	out = append(out, fmt.Sprintf("Namespace:    %s", pod.Namespace))
	out = append(out, fmt.Sprintf("Priority:     %d", *pod.Spec.Priority))
	out = append(out, fmt.Sprintf("Node:         %s", pod.Spec.NodeName))
	if pod.Status.StartTime != nil {
		out = append(out, fmt.Sprintf("Start Time:   %s", pod.Status.StartTime.Time))
	}

	out = append(out, fmt.Sprintf("Phase:        %s", pod.Status.Phase))

	out = append(out, misc.RenderMapOfStrings("Labels:", pod.Labels)...)
	out = append(out, misc.RenderMapOfStrings("Annotations:", pod.Annotations)...)

	out = append(out, "\nConditions:")
	for _, condition := range pod.Status.Conditions {
		out = append(out, fmt.Sprintf("  Type: %v", condition.Type))
		out = append(out, fmt.Sprintf("    Status: %v", condition.Status))
		out = append(out, fmt.Sprintf("    Reason: %v", condition.Reason))
		if condition.Message != "" {
			out = append(out, fmt.Sprintf("    Message: %v", condition.Message))
		}
	}

	out = append(out, "\nContainers:")
	for _, container := range pod.Spec.Containers {
		out = append(out, fmt.Sprintf("  %s:", container.Name))
		out = append(out, fmt.Sprintf("    Image:           %s", container.Image))
		out = append(out, fmt.Sprintf("    Ports:           %v", container.Ports))
		out = append(out, fmt.Sprintf("    Host Ports:      %v", container.Ports))
		out = append(out, fmt.Sprintf("    Resource Limits: %v", container.Resources.Limits))
		out = append(out, fmt.Sprintf("    Liveness:        %v", container.LivenessProbe))
		out = append(out, fmt.Sprintf("    Readiness:       %v", container.ReadinessProbe))
	}

	out = append(out, "\nStatus:")
	for _, containerStatus := range pod.Status.ContainerStatuses {
		out = append(out, fmt.Sprintf("  %s:", containerStatus.Name))
		out = append(out, fmt.Sprintf("    Ready: %v", containerStatus.Ready))
		out = append(out, fmt.Sprintf("    Restart Count: %d", containerStatus.RestartCount))
		out = append(out, fmt.Sprintf("    Image: %s", containerStatus.Image))
		out = append(out, fmt.Sprintf("    Image ID: %s", containerStatus.ImageID))
		out = append(out, fmt.Sprintf("    Container ID: %s", containerStatus.ContainerID))
	}

	return out
}
