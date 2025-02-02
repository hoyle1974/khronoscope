package format

import (
	"fmt"
	"strings"

	"github.com/hoyle1974/khronoscope/internal/misc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Ports(ports []corev1.ContainerPort) string {
	var portStrings []string
	for _, port := range ports {
		portStrings = append(portStrings, fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol))
	}
	return strings.Join(portStrings, ", ")
}

func Args(args []string) string {
	return strings.Join(args, " ")
}

func Limits(limits map[corev1.ResourceName]resource.Quantity) string {
	var limitStrings []string
	for resource, quantity := range misc.Range(limits) {
		limitStrings = append(limitStrings, fmt.Sprintf("%s: %s", resource, quantity.String()))
	}
	return strings.Join(limitStrings, " ")
}

func Liveness(probe *corev1.Probe) string {
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

func Environment(envVars []corev1.EnvVar) string {
	var envStrings []string
	for _, env := range envVars {
		envStrings = append(envStrings, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return strings.Join(envStrings, ", ")
}

func VolumeMounts(mounts []corev1.VolumeMount) string {
	var mountStrings []string
	for _, mount := range mounts {
		mountStrings = append(mountStrings, fmt.Sprintf("%s from %s (%t)", mount.MountPath, mount.Name, mount.ReadOnly))
	}
	return strings.Join(mountStrings, ", ")
}

func NodeSelectors(nodeSelectors map[string]string) string {
	var selectorStrings []string
	for key, value := range misc.Range(nodeSelectors) {
		selectorStrings = append(selectorStrings, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(selectorStrings, " ")
}

func Tolerations(tolerations []corev1.Toleration) string {
	var tolerationStrings []string
	for _, tol := range tolerations {
		tolerationStrings = append(tolerationStrings, fmt.Sprintf("%s %s=%s:%s", tol.Operator, tol.Key, tol.Value, tol.Effect))
	}
	return strings.Join(tolerationStrings, " ")
}

func VolumeSource(source corev1.VolumeSource) string {
	if source.HostPath != nil {
		return fmt.Sprintf("HostPath (path: %s)", source.HostPath.Path)
	}
	return "<unknown>"
}
