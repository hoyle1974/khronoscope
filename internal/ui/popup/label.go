package popup

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
			return p, Close
		case tea.KeyEnter:
			// Save the label
			p.labeler.SetLabel(p.textInput.Value())
			return p, Close
		}
	}

	p.textInput, _ = p.textInput.Update(msg)

	return p, nil
}

func (model *labelPopupModel) View() string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		BorderStyle(b).
		Padding(1).
		Width(model.width - 2).
		Height(5).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return style.Render(fmt.Sprintf(
		"Add a label to this timestamp\n\n%s\n\n%s",
		model.textInput.View(),
		"(esc to quit)",
	))
}

func (p *labelPopupModel) OnResize(width, height int) {
	p.width = width
	p.height = height
}

func NewLabelPopup(labeler Labeler) Popup {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return &labelPopupModel{textInput: ti, labeler: labeler}
}
