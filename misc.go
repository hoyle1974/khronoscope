package main

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// renderProgressBar generates a 12-character progress bar with percentage display
func renderProgressBar(label string, percent float64) string {
	// Ensure percent is within 0-100
	if percent < 0 {
		percent = 0
	} else if percent > 100 {
		percent = 100
	}

	// Calculate filled segments (10 total)
	filledSegments := int(math.Round(percent / 10))

	r := 0
	g := 204
	b := 0
	s := fmt.Sprintf("#%02x%02x%02x%02x", r, g, b, 255)
	cc := lipgloss.Color(s)
	if percent > 90 {
		pp := (((percent - 90.0) * 10.0) / 100.0)
		r := int(255 * pp)
		g := 204 - int(204*pp)
		b := 0
		s := fmt.Sprintf("#%02x%02x%02x%02x", r, g, b, 255)
		cc = lipgloss.Color(s)
	}

	// Define styles for filled and empty segments
	filledStyle := lipgloss.NewStyle().Background(cc).Foreground(lipgloss.Color("#000000"))                   // Green
	emptyStyle := lipgloss.NewStyle().Background(lipgloss.Color("240")).Foreground(lipgloss.Color("#FFFFFF")) // Gray

	// Format percentage to fit within 3 characters
	percentText := fmt.Sprintf("      %3.2f%%", percent)

	// Build the bar
	bar := ""
	for i := 0; i < 10; i++ {
		if i < filledSegments {
			bar += filledStyle.Render(string(percentText[i]))
		} else {
			bar += emptyStyle.Render(string(percentText[i]))
		}
	}
	// bar += "]"

	// Overlay percentage text
	return label + " " + bar
}

func RenderMapOfStrings(name string, t map[string]string) []string {
	out := []string{}

	out = append(out, name)

	for k, v := range NewMapRangeFunc(t) {
		out = append(out, fmt.Sprintf("   %v : %v", k, v))
	}
	return out
}

func NewMapRangeFunc[K comparable, V any](m map[K]V) func(func(K, V) bool) {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	// Sort the keys dynamically based on their type.
	sort.Slice(keys, func(i, j int) bool {
		return compareKeys(keys[i], keys[j]) < 0
	})

	// Return a function that takes a callback to process each key-value pair.
	return func(callback func(K, V) bool) {
		for _, key := range keys {
			value := m[key]
			// If the callback returns false, stop iteration.
			if !callback(key, value) {
				break
			}
		}
	}
}

// compareKeys determines how to sort keys dynamically.
func compareKeys[K comparable](a, b K) int {
	// Use reflection to determine the type of the key.
	kind := reflect.TypeOf(a).Kind()
	switch kind {
	case reflect.String:
		// Compare strings lexicographically.
		strA := fmt.Sprintf("%v", a)
		strB := fmt.Sprintf("%v", b)
		if strA < strB {
			return -1
		} else if strA > strB {
			return 1
		}
		return 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		// Compare numbers numerically.
		numA, numB := reflect.ValueOf(a).Float(), reflect.ValueOf(b).Float()
		if numA < numB {
			return -1
		} else if numA > numB {
			return 1
		}
		return 0
	default:
		// Convert other types to strings and compare lexicographically.
		strA, strB := fmt.Sprintf("%v", a), fmt.Sprintf("%v", b)
		if strA < strB {
			return -1
		} else if strA > strB {
			return 1
		}
		return 0
	}
}

func formatPorts(ports []corev1.ContainerPort) string {
	var portStrings []string
	for _, port := range ports {
		portStrings = append(portStrings, fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol))
	}
	return strings.Join(portStrings, ", ")
}

func formatArgs(args []string) string {
	return strings.Join(args, " ")
}

func formatLimits(limits map[corev1.ResourceName]resource.Quantity) string {
	var limitStrings []string
	for resource, quantity := range NewMapRangeFunc(limits) {
		limitStrings = append(limitStrings, fmt.Sprintf("%s: %s", resource, quantity.String()))
	}
	return strings.Join(limitStrings, " ")
}

func formatLiveness(probe *corev1.Probe) string {
	if probe == nil {
		return "<none>"
	}
	return fmt.Sprintf("http-get %s delay=%s timeout=%s period=%s #success=%d #failure=%d", probe.HTTPGet.Path, probe.InitialDelaySeconds, probe.TimeoutSeconds, probe.PeriodSeconds, probe.SuccessThreshold, probe.FailureThreshold)
}

func formatEnvironment(envVars []corev1.EnvVar) string {
	var envStrings []string
	for _, env := range envVars {
		envStrings = append(envStrings, fmt.Sprintf("%s=%s", env.Name, env.Value))
	}
	return strings.Join(envStrings, ", ")
}

func formatVolumeMounts(mounts []corev1.VolumeMount) string {
	var mountStrings []string
	for _, mount := range mounts {
		mountStrings = append(mountStrings, fmt.Sprintf("%s from %s (%s)", mount.MountPath, mount.Name, mount.ReadOnly))
	}
	return strings.Join(mountStrings, ", ")
}

func formatNodeSelectors(nodeSelectors map[string]string) string {
	var selectorStrings []string
	for key, value := range NewMapRangeFunc(nodeSelectors) {
		selectorStrings = append(selectorStrings, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(selectorStrings, " ")
}

func formatTolerations(tolerations []corev1.Toleration) string {
	var tolerationStrings []string
	for _, tol := range tolerations {
		tolerationStrings = append(tolerationStrings, fmt.Sprintf("%s %s=%s:%s", tol.Operator, tol.Key, tol.Value, tol.Effect))
	}
	return strings.Join(tolerationStrings, " ")
}
