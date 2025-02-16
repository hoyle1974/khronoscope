package popup

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// type Labeler interface {
// 	SetLabel(label string)
// }

type savePopupModel struct {
	textInput     textinput.Model
	onDone        func(string)
	width, height int
}

func (model *savePopupModel) Init() tea.Cmd { return nil }

func (model *savePopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			model.onDone("")
			return model, Close
		case tea.KeyEnter:
			// Save the label
			model.onDone(model.textInput.Value())
			return model, Close
		}
	}

	model.textInput, _ = model.textInput.Update(msg)

	return model, nil
}

func (model *savePopupModel) View() string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		BorderStyle(b).
		Padding(1).
		Width(model.width - 2).
		Height(5).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return style.Render(fmt.Sprintf(
		"Save data:\n\n%s\n\n%s",
		model.textInput.View(),
		"(esc to quit)",
	))
}

func (model *savePopupModel) OnResize(width, height int) {
	model.width = width
	model.height = height
}

func NewSavePopup(onDone func(string)) Popup {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20
	ti.SetValue("session.khron")

	return &savePopupModel{textInput: ti, onDone: onDone}
}
