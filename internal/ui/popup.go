package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PopupView handles the rendering of popup windows
type PopupView struct {
	width  int
	height int
}

// NewPopupView creates a new popup view instance
func NewPopupView(width, height int) *PopupView {
	return &PopupView{
		width:  width,
		height: height,
	}
}

// Update updates the popup dimensions
func (p *PopupView) Update(width, height int) {
	p.width = width
	p.height = height
}

// InsertPopup renders a popup window with the given content
func (p *PopupView) InsertPopup(content string, popup interface{ Render() string }) string {
	popupContent := popup.Render()
	lines := strings.Split(content, "\n")

	popupLines := strings.Split(popupContent, "\n")
	popupHeight := len(popupLines)
	popupWidth := 0
	for _, line := range popupLines {
		if len(line) > popupWidth {
			popupWidth = len(line)
		}
	}

	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1, 0).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	popupBox := popupStyle.Render(popupContent)
	popupBoxLines := strings.Split(popupBox, "\n")

	startRow := (p.height - popupHeight) / 2
	startCol := (p.width - popupWidth) / 2

	result := make([]string, len(lines))
	for i := 0; i < len(lines); i++ {
		if i >= startRow && i < startRow+len(popupBoxLines) {
			popupLine := popupBoxLines[i-startRow]
			prefix := lines[i][:startCol]
			suffix := lines[i][startCol+len(popupLine):]
			result[i] = prefix + popupLine + suffix
		} else {
			result[i] = lines[i]
		}
	}

	return strings.Join(result, "\n")
}
