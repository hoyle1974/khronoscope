package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
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
