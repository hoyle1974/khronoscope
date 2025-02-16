package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/internal/misc"
)

type debugPopupModel struct {
	width      int
	height     int
	program    *tea.Program
	ringBuffer *misc.RingBuffer
}

func NewDebugPopupModel(program *tea.Program, ringBuffer *misc.RingBuffer) Popup {
	return &debugPopupModel{
		program:    program,
		ringBuffer: ringBuffer,
	}
}

func (p *debugPopupModel) OnResize(width, height int) {
	p.width = width
	p.height = height
}

func (p *debugPopupModel) Init() tea.Cmd {
	return nil
}

func (p *debugPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return p, Close
		}
	}

	return p, nil
}

func (p *debugPopupModel) Close() {
	p.program.Send(PopupClose{})
}

func (p *debugPopupModel) View() string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		BorderStyle(b).
		Padding(0).
		Width(p.width - 2).
		Height(p.height - 2).
		AlignHorizontal(lipgloss.Left).
		AlignVertical(lipgloss.Top)

	return style.Render(p.ringBuffer.String())
}
