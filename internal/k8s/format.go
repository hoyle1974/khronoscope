package k8s

import (
	"fmt"
	"strings"

	"github.com/hoyle1974/khronoscope/internal/misc"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// FormatPorts formats container ports into a readable string
func FormatPorts(ports []corev1.ContainerPort) string {
	var portStrings []string
	for _, port := range ports {
		portStrings = append(portStrings, fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol))
	}
	return strings.Join(portStrings, ", ")
}

// FormatArgs formats command arguments into a readable string
func FormatArgs(args []string) string {
	return strings.Join(args, " ")
}

// FormatLimits formats resource limits into a readable string
func FormatLimits(limits map[corev1.ResourceName]resource.Quantity) string {
	var limitStrings []string
	for resource, quantity := range misc.Range(limits) {
		limitStrings = append(limitStrings, fmt.Sprintf("%s: %s", resource, quantity.String()))
	}
	return strings.Join(limitStrings, " ")
}

// FormatLiveness formats a probe into a readable string
func FormatLiveness(probe *corev1.Probe) string {
	if probe == nil {
		return "<none>"
	}
	return fmt.Sprintf("http-get %s delay=%d timeout=%d period=%d #success=%d #failure=%d",
		probe.HTTPGet.Path,
		probe.InitialDelaySeconds,
		probe.TimeoutSeconds,
		probe.PeriodSeconds,
		probe.SuccessThreshold,
		probe.FailureThreshold)
}

// FormatEnvironment formats environment variables into a readable string
func FormatEnvironment(envVars []corev1.EnvVar) string {
	var envStrings []string
	for _, env := range envVars {
		envStrings = append(envStrings, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return strings.Join(envStrings, ", ")
}

// FormatVolumeMounts formats volume mounts into a readable string
func FormatVolumeMounts(mounts []corev1.VolumeMount) string {
	var mountStrings []string
	for _, mount := range mounts {
		mountStrings = append(mountStrings, fmt.Sprintf("%s from %s (%t)", mount.MountPath, mount.Name, mount.ReadOnly))
	}
	return strings.Join(mountStrings, ", ")
}

// FormatNodeSelectors formats node selectors into a readable string
func FormatNodeSelectors(nodeSelectors map[string]string) string {
	var selectorStrings []string
	for key, value := range misc.Range(nodeSelectors) {
		selectorStrings = append(selectorStrings, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(selectorStrings, " ")
}

// FormatTolerations formats tolerations into a readable string
func FormatTolerations(tolerations []corev1.Toleration) string {
	var tolerationStrings []string
	for _, tol := range tolerations {
		tolerationStrings = append(tolerationStrings, fmt.Sprintf("%s %s=%s:%s", tol.Operator, tol.Key, tol.Value, tol.Effect))
	}
	return strings.Join(tolerationStrings, " ")
}

// FormatVolumeSource formats a volume source into a readable string
func FormatVolumeSource(source corev1.VolumeSource) string {
	if source.HostPath != nil {
		return fmt.Sprintf("HostPath (path: %s)", source.HostPath.Path)
	}
	return "<unknown>"
}

// FormatLabelSelector formats a label selector into a readable string
func FormatLabelSelector(selector *metav1.LabelSelector) string {
	if selector == nil {
		return "<none>"
	}

	var requirements []string

	// Add match labels
	for key, value := range selector.MatchLabels {
		requirements = append(requirements, fmt.Sprintf("%s=%s", key, value))
	}

	// Add match expressions
	for _, expr := range selector.MatchExpressions {
		switch expr.Operator {
		case metav1.LabelSelectorOpIn:
			requirements = append(requirements, fmt.Sprintf("%s in (%s)", expr.Key, strings.Join(expr.Values, ",")))
		case metav1.LabelSelectorOpNotIn:
			requirements = append(requirements, fmt.Sprintf("%s notin (%s)", expr.Key, strings.Join(expr.Values, ",")))
		case metav1.LabelSelectorOpExists:
			requirements = append(requirements, expr.Key)
		case metav1.LabelSelectorOpDoesNotExist:
			requirements = append(requirements, fmt.Sprintf("!%s", expr.Key))
		}
	}

	if len(requirements) == 0 {
		return "<none>"
	}

	return strings.Join(requirements, ",")
}
