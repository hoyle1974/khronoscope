package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/types"
)

type ContainerSelect interface {
	SetContainer(name string)
}

type containerPopupModel struct {
	Containers        []string
	Select            int
	OnContainerSelect func(string)
}

func (p *containerPopupModel) Update(msg tea.Msg) bool {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			p.Up()
			return false
		case tea.KeyDown:
			p.Down()
			return false
		case tea.KeyCtrlC, tea.KeyEsc:
			return true
		case tea.KeyEnter:
			// Save the label
			p.OnContainerSelect(p.Containers[p.Select])
			return true
		}
	}

	return false
}

func (model *containerPopupModel) Up() {
	model.Select--
	if model.Select < 0 {
		model.Select += len(model.Containers)
	}
}

func (model *containerPopupModel) Down() {
	model.Select++
	if model.Select >= len(model.Containers) {
		model.Select %= len(model.Containers)
	}
}

func (model *containerPopupModel) View(width, height int) string {
	return RenderContainerPopup(model, width, height)
}

func NewContainerPopupModel(client conn.KhronosConn, resource types.Resource, selector func(string)) Popup {
	if resource.GetKind() != "Pod" {
		return nil
	}

	extra := resource.GetExtra().(resources.PodExtra)

	containers := []string{}
	for k := range extra.Containers {
		containers = append(containers, k)
	}
	if len(containers) == 0 {
		return nil
	}
	if len(containers) == 1 {
		selector(containers[0])
		return nil
	}

	return &containerPopupModel{Containers: containers, OnContainerSelect: selector}
}

func RenderContainerPopup(model *containerPopupModel, width, height int) string {
	b := lipgloss.RoundedBorder()
	style := lipgloss.NewStyle().
		BorderStyle(b).
		Padding(1).
		Width(width - 2).
		Height(5).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	sel := lipgloss.NewStyle().Bold(true)

	c := ""
	for idx, container := range model.Containers {
		if model.Select == idx {
			c += sel.Render(container) + "\n"
		} else {
			c += container + "\n"
		}
	}

	return style.Render(fmt.Sprintf(
		"Select a container:\n\n%s\n%s",
		c,
		"(esc to quit)",
	))
}
