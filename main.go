package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()

	bold = lipgloss.NewStyle().Bold(true)
)

func main() {
	// Connect to kubernetes
	client, err := NewKhronosConnection()
	if err != nil {
		fmt.Errorf("Error creating conneciton: %w", err)
		return
	}

	var watcher = NewK8sWatcher()

	watchForDeployments(watcher, client)
	watchForDaemonSet(watcher, client)
	watchForReplicaSet(watcher, client)
	watchForService(watcher, client)
	watchForNamespaces(watcher, client)
	podWatchMe := watchForPods(watcher, client)
	watchForNodes(watcher, client, podWatchMe)

	p := tea.NewProgram(
		newModel(watcher),
	)

	watcher.OnChange(func() {
		p.Send(1)
	})

	if err := p.Start(); err != nil {
		panic(err)
	}

}
