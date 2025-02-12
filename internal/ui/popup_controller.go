package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Popup interface {
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
	Init() tea.Cmd
}

type Labeler interface {
	SetLabel(label string)
}

type labelPopupModel struct {
	textInput     textinput.Model
	labeler       Labeler
	width, height int
}

func (p *labelPopupModel) Init() tea.Cmd { return nil }

func (p *labelPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return p, tea.Quit
		case tea.KeyEnter:
			// Save the label
			p.labeler.SetLabel(p.textInput.Value())
			return p, tea.Quit
		}
	}

	p.textInput, _ = p.textInput.Update(msg)

	return p, nil
}

func (model *labelPopupModel) View() string {
	return RenderLabelPopup(model, model.width, model.height)
}

func NewLabelPopup(width, height int, labeler Labeler) Popup {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return &labelPopupModel{textInput: ti, labeler: labeler, width: width, height: height}
}

type messagePopupModel struct {
	msg           string
	close         string
	width, height int
}

func (p *messagePopupModel) Init() tea.Cmd { return nil }

func (p *messagePopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case p.close:
			return p, tea.Quit
		}
	}

	return p, nil
}

func (model *messagePopupModel) View() string {
	return RenderMessagePopup(model, model.width, model.height)
}

// NewMessagePopup creates a new message popup with the given message and close key
func NewMessagePopup(msg string, close string, width, height int) Popup {
	return &messagePopupModel{msg: msg, close: close, width: width, height: height}
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
