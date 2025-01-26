package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Popup interface {
	Update(msg tea.Msg) bool
	View(width, height int) string
}

type Labeler interface {
	SetLabel(label string)
}

type labelPopupModel struct {
	textInput textinput.Model
	labeler   Labeler
}

func (p *labelPopupModel) Update(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return true
		case tea.KeyEnter:
			// Save the label
			p.labeler.SetLabel(p.textInput.Value())
			return true
		}
	}

	p.textInput, _ = p.textInput.Update(msg)

	return false
}

func (model *labelPopupModel) View(width, height int) string {
	return RenderLabelPopup(model, width, height)
}

func NewLabelPopup(labeler Labeler) Popup {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return &labelPopupModel{textInput: ti, labeler: labeler}
}

type messagePopupModel struct {
	msg   string
	close string
}

func (p *messagePopupModel) Update(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case p.close:
			return true
		}
	}

	return false
}

func (model *messagePopupModel) View(width, height int) string {
	return RenderMessagePopup(model, width, height)
}

// NewMessagePopup creates a new message popup with the given message and close key
func NewMessagePopup(msg string, close string) Popup {
	return &messagePopupModel{msg: msg, close: close}
}

func RenderLabelPopup(model *labelPopupModel, width, height int) string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		BorderStyle(b).
		Padding(1).
		Width(width - 2).
		Height(5).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return style.Render(fmt.Sprintf(
		"Add a label to this timestamp\n\n%s\n\n%s",
		model.textInput.View(),
		"(esc to quit)",
	))
}

func RenderMessagePopup(model *messagePopupModel, width, height int) string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		BorderStyle(b).
		Padding(1).
		Width(width - 2).
		Height(3).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return style.Render(model.msg)
}
