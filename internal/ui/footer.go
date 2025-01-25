package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FooterView handles the rendering of the application footer
type FooterView struct {
	width         int
	scrollPercent float64
}

// NewFooterView creates a new footer view instance
func NewFooterView(width int) *FooterView {
	return &FooterView{
		width: width,
	}
}

// Update updates the footer view state
func (f *FooterView) Update(width int, scrollPercent float64) {
	f.width = width
	f.scrollPercent = scrollPercent
}

// Render renders the footer view
func (f *FooterView) Render() string {
	info := fmt.Sprintf(" %3.f%%", f.scrollPercent*100)
	line := strings.Repeat("─", max(0, f.width-len(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
