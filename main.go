package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {

	client, err := NewKhronosConnection()
	if err != nil {
		fmt.Printf("Error creating connection: %v", err)
		return
	}

	var watcher = NewK8sWatcher()

	watchForDeployments(watcher, client)
	watchForDaemonSet(watcher, client)
	watchForReplicaSet(watcher, client)
	watchForService(watcher, client)
	watchForNamespaces(watcher, client)
	podWatcher := watchForPods(watcher, client)
	watchForNodes(watcher, client, podWatcher)

	appModel := newModel(watcher)
	p := tea.NewProgram(appModel)

	appModel.vcr = NewVCRControl(watcher.temporalMap, func() {
		p.Send(1)
	})

	watcher.OnChange(func() {
		p.Send(1)
	})

	if err := p.Start(); err != nil {
		panic(err)
	}

}
