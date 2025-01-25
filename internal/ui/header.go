package ui

import (
	"fmt"
	"math"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// HeaderView handles the rendering of the application header
type HeaderView struct {
	minTime     time.Time
	maxTime     time.Time
	currentTime time.Time
	vcrEnabled  bool
}

// NewHeaderView creates a new header view instance
func NewHeaderView() *HeaderView {
	return &HeaderView{}
}

// Update updates the header view state
func (h *HeaderView) Update(minTime, maxTime, currentTime time.Time, vcrEnabled bool) {
	h.minTime = minTime
	h.maxTime = maxTime
	h.currentTime = currentTime
	h.vcrEnabled = vcrEnabled
}

// calculatePercentageOfTime calculates the percentage of time elapsed
func (h *HeaderView) calculatePercentageOfTime() float64 {
	if h.currentTime.Before(h.minTime) || h.currentTime.After(h.maxTime) {
		return 0
	}

	minUnix := h.minTime.Unix()
	maxUnix := h.maxTime.Unix()
	valueUnix := h.currentTime.Unix()

	return float64(valueUnix-minUnix) / float64(maxUnix-minUnix)
}

// Render renders the header view
func (h *HeaderView) Render() string {
	currentTime := fmt.Sprintf(" Current Time: %s ", h.currentTime.Format("2006-01-02 15:04:05"))

	if !h.vcrEnabled {
		return currentTime
	}

	p := h.calculatePercentageOfTime()
	percentText := fmt.Sprintf("Available Range (%s to %s) %3.2f%% ",
		h.minTime.Format("2006-01-02 15:04:05"),
		h.maxTime.Format("2006-01-02 15:04:05"),
		p*100,
	)

	size := len(percentText)
	filledSegments := int(math.Round(p * float64(size)))

	filledStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000"))
	emptyStyle := lipgloss.NewStyle().Background(lipgloss.Color("#0000FF")).Foreground(lipgloss.Color("#FFFFFF"))

	bar := currentTime + " ["
	for i := 0; i < len(percentText); i++ {
		if i < filledSegments {
			bar += filledStyle.Render(string(percentText[i]))
		} else {
			bar += emptyStyle.Render(string(percentText[i]))
		}
	}
	bar += "]"

	return bar
}
