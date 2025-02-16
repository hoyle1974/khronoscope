package popup

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
			return p, Close
		}
	}

	return p, nil
}

func (model *messagePopupModel) View() string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		BorderStyle(b).
		Padding(1).
		Width(model.width - 2).
		Height(3).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return style.Render(model.msg)
}

// NewMessagePopup creates a new message popup with the given message and close key
func NewMessagePopup(msg string, close string, width, height int) Popup {
	return &messagePopupModel{msg: msg, close: close, width: width, height: height}
}

// func RenderMessagePopup(model *messagePopupModel, width, height int) string {
// 	b := lipgloss.RoundedBorder()
// 	style := lipgloss.NewStyle().
// 		BorderStyle(b).
// 		Padding(1).
// 		Width(width - 2).
// 		Height(3).
// 		AlignHorizontal(lipgloss.Center).
// 		AlignVertical(lipgloss.Center)

// 	return style.Render(model.msg)
// }
