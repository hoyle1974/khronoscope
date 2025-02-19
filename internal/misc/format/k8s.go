package format

import (
	"fmt"

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
