package popup

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
	paused     bool
	lastBuffer string
}

func NewDebugPopupModel(program *tea.Program, ringBuffer *misc.RingBuffer) Popup {
	return &debugPopupModel{
		program:    program,
		ringBuffer: ringBuffer,
	}
}

func (model *debugPopupModel) OnResize(width, height int) {
	model.width = width
	model.height = height
}

func (model *debugPopupModel) Init() tea.Cmd {
	return nil
}

func (model *debugPopupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return model, Close
		case tea.KeySpace:
			model.paused = !model.paused
			if model.paused {
				model.lastBuffer = model.ringBuffer.String() + "{paused}"
			}
			return model, nil
		}
	}

	return model, nil
}

func (p *debugPopupModel) Close() {
	p.program.Send(PopupClose{})
}

func (model *debugPopupModel) View() string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		BorderStyle(b).
		Padding(0).
		Width(model.width - 2).
		Height(model.height - 2).
		AlignHorizontal(lipgloss.Left).
		AlignVertical(lipgloss.Top)

	if !model.paused {
		model.lastBuffer = model.ringBuffer.String()
	}

	return style.Render(model.lastBuffer)
}
