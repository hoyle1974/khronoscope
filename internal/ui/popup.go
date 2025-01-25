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

type LabelPopup struct {
	textInput textinput.Model
	labeler   Labeler
}

func (p *LabelPopup) Update(msg tea.Msg) bool {
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

func (p *LabelPopup) View(width, height int) string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().BorderStyle(b).Padding(1).Width(width - 2).Height(3).AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center)

	return style.Render(fmt.Sprintf(
		"Add a label to this timestamp\n\n%s\n\n%s",
		p.textInput.View(),
		"(esc to quit)",
	) + "\n")
}

func NewLabelPopup(labeler Labeler) Popup {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return &LabelPopup{textInput: ti, labeler: labeler}
}

type MessagePopup struct {
	msg   string
	close string
}

func (p *MessagePopup) Update(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case p.close:
			return true
		}
	}

	return false
}

func (p *MessagePopup) View(width, height int) string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().BorderStyle(b).Padding(1).Width(width - 2).Height(3).AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center)
	return style.Render(p.msg)
}

// NewMessagePopup creates a new message popup with the given message and close key
func NewMessagePopup(msg string, close string) Popup {
	return &MessagePopup{msg: msg, close: close}
}
