package main

import (
	"encoding/gob"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	gob.Register(Resource{})
	gob.Register(TemporalMap{})
	gob.Register(ReplicaSetExtra{})
	gob.Register(NodeExtra{})
	gob.Register(PodExtra{})

	client, err := NewKhronosConnection()
	if err != nil {
		fmt.Printf("Error creating connection: %v", err)
		return
	}

	// filename := "temp.dat"
	filename := ""

	data := NewDataModel()

	if len(filename) > 0 {
		data = NewDataModelFromFile(filename)
	}
	var watcher = NewK8sWatcher(data)

	if len(filename) > 0 {
		watcher = nil
	}

	watchForDeployments(watcher, client)
	watchForDaemonSet(watcher, client)
	watchForReplicaSet(watcher, client)
	watchForService(watcher, client)
	watchForNamespaces(watcher, client)
	podWatcher := watchForPods(watcher, client, data)
	watchForNodes(watcher, client, data, podWatcher)

	appModel := newModel(watcher, data)
	p := tea.NewProgram(appModel)

	appModel.vcr = NewVCRControl(data, func() {
		p.Send(1)
	})

	if len(filename) > 0 {
		min, _ := data.GetTimeRange()
		appModel.vcr.enableVCR()
		appModel.vcr.vcrTime = min
	}

	watcher.OnChange(func() {
		p.Send(1)
	})

	if _, err := p.Run(); err != nil {
		panic(err)
	}

}
