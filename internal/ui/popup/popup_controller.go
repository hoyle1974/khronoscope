package popup

import (
	tea "github.com/charmbracelet/bubbletea"
)

type PopupClose struct{}

func Close() tea.Msg {
	return PopupClose{}
}

type Popup interface {
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
	Init() tea.Cmd
}
