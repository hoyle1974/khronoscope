package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Popup interface {
	Update(msg tea.Msg) bool
	View(width, height int) string
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

func newMessagePopup(msg string, close string) Popup {
	return &MessagePopup{msg: msg, close: close}
}
