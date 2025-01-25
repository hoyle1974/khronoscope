package misc

import (
	"fmt"
	"math"

	"github.com/charmbracelet/lipgloss"
)

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
	// bar += "]"

	// Overlay percentage text
	return label + " " + bar
}

func RenderMapOfStrings(name string, t map[string]string) []string {
	out := []string{}

	out = append(out, name)

	for k, v := range Range(t) {
		out = append(out, fmt.Sprintf("   %v : %v", k, v))
	}
	return out
}
