package main

import (
	"encoding/gob"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hoyle1974/khronoscope/conn"
	"github.com/hoyle1974/khronoscope/internal/ui"
	"github.com/hoyle1974/khronoscope/resources"
)

func main() {
	gob.Register(resources.Resource{})
	gob.Register(resources.ReplicaSetExtra{})
	gob.Register(resources.NodeExtra{})
	gob.Register(resources.PodExtra{})

	client, err := conn.NewKhronosConnection()
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
	var watcher = resources.NewK8sWatcher(data)

	if len(filename) > 0 {
		watcher = nil
	}

	watcher.Watch(client, data)

	appModel := newModel(watcher, data)
	p := tea.NewProgram(appModel)

	appModel.vcr = ui.NewVCRControl(data, func() {
		p.Send(1)
	})

	if len(filename) > 0 {
		min, _ := data.GetTimeRange()
		appModel.vcr.EnableVCR()
		appModel.vcr.SetTime(min)
	}

	watcher.OnChange(func() {
		p.Send(1)
	})

	if _, err := p.Run(); err != nil {
		panic(err)
	}

}
