package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/resources"
)

// DetailView handles the rendering of resource details
type DetailView struct {
	viewport viewport.Model
	width    int
	height   int
	resource *resources.Resource
}

// NewDetailView creates a new detail view instance
func NewDetailView(width, height int) *DetailView {
	return &DetailView{
		width:    width,
		height:   height,
		viewport: viewport.New(width, height),
	}
}

// Update updates the detail view state
func (dv *DetailView) Update(width, height int, resource *resources.Resource) {
	dv.width = width
	dv.height = height
	dv.resource = resource

	dv.viewport.Width = width
	dv.viewport.Height = height

	dv.viewport.SetContent(dv.renderContent())
}

// renderContent generates the detail view content
func (dv *DetailView) renderContent() string {
	if dv.resource == nil {
		return "No resource selected"
	}

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true)

	header := style.Render(fmt.Sprintf("%s/%s (%s)\n",
		dv.resource.Namespace,
		dv.resource.Name,
		dv.resource.Kind,
	))

	// Get the details using the resource's renderer
	details := dv.resource.GetDetails()

	return header + strings.Join(details, "\n")
}

// View renders the detail view
func (dv *DetailView) View() string {
	return dv.viewport.View()
}

// ScrollUp scrolls the view up
func (dv *DetailView) ScrollUp() {
	dv.viewport.LineUp(1)
}

// ScrollDown scrolls the view down
func (dv *DetailView) ScrollDown() {
	dv.viewport.LineDown(1)
}

// PageUp scrolls the view up one page
func (dv *DetailView) PageUp() {
	dv.viewport.HalfViewUp()
}

// PageDown scrolls the view down one page
func (dv *DetailView) PageDown() {
	dv.viewport.HalfViewDown()
}

// LineUp scrolls the view up by n lines
func (dv *DetailView) LineUp(n int) {
	dv.viewport.LineUp(n)
}

// LineDown scrolls the view down by n lines
func (dv *DetailView) LineDown(n int) {
	dv.viewport.LineDown(n)
}
