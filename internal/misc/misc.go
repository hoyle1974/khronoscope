package misc

import (
	"fmt"
	"math"

	"github.com/charmbracelet/lipgloss"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

func ConvertArrayToSet[V comparable](arr []V) map[V]any {
	set := make(map[V]any)
	for _, val := range arr {
		set[val] = true
	}
	return set
}

// formatCreationTimestamp ensures the timestamp is human-readable
func FormatCreationTimestamp(timestamp *v1.Time) string {
	if timestamp == nil {
		return "<none>"
	}
	return duration.HumanDuration(v1.Now().Sub(timestamp.Time))
}

func FormatNilString(arr *string) string {
	if arr == nil || len(*arr) == 0 {
		return "<none>"
	}
	return *arr
}

// formatNilArray returns "<none>" if the array is nil or empty, otherwise it formats it as a comma-separated string
func FormatNilArray(arr []string) string {
	if len(arr) == 0 {
		return "<none>"
	}
	return FormatArray(arr)
}

// formatArray converts an array into a comma-separated string
func FormatArray(arr []string) string {
	if len(arr) == 0 {
		return "<none>"
	}
	if len(arr) == 1 {
		return arr[0]
	}
	return fmt.Sprintf("[%s]", arr)
}

func DeepCopyArray[K any](s []K) []K {
	dest := make([]K, len(s))

	for k := 0; k < len(s); k++ {
		dest[k] = deepCopyValue(s[k])
	}

	return dest
}

func DeepCopyMap[K comparable, V any](m map[K]V) map[K]V {
	newMap := make(map[K]V, len(m))

	for k, v := range m {
		newMap[k] = deepCopyValue(v)
	}

	return newMap
}

func deepCopyValue[V any](v V) V {
	switch v := any(v).(type) {
	case map[any]any:
		return any(DeepCopyMap(v)).(V)
	case []any:
		return any(deepCopySlice(v)).(V)
	default:
		return v.(V)
	}
}

func deepCopySlice[V any](s []V) []V {
	newSlice := make([]V, len(s))
	for i, v := range s {
		newSlice[i] = deepCopyValue(v)
	}
	return newSlice
}

// renderProgressBar generates a 12-character progress bar with percentage display
func RenderProgressBar(label string, percent float64) string {
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

	// Overlay percentage text
	return label + " " + bar
}

func RenderMapOfStrings[V any](t map[string]V) []string {
	out := []string{}
	if len(t) == 0 {
		out = append(out, "   <none>")
	}

	for k, v := range Range(t) {
		out = append(out, fmt.Sprintf("   %v : %v", k, v))
	}
	return out
}
