package main

import (
	"encoding/gob"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hoyle1974/khronoscope/conn"
	"github.com/hoyle1974/khronoscope/internal/app"
	"github.com/hoyle1974/khronoscope/internal/data"
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

	d := data.New()

	if len(filename) > 0 {
		d = data.NewFromFile(filename)
	}
	var watcher = resources.NewK8sWatcher(d)

	if len(filename) > 0 {
		watcher = nil
	}

	watcher.Watch(client, d)

	appModel := app.NewAppModel(watcher, d)
	p := tea.NewProgram(appModel)

	appModel.VCR = ui.NewTimeController(d, func() {
		p.Send(1)
	})

	if len(filename) > 0 {
		min, _ := d.GetTimeRange()
		appModel.VCR.EnableVirtualTime()
		appModel.VCR.SetTime(min)
	}

	watcher.OnChange(func() {
		p.Send(1)
	})

	if _, err := p.Run(); err != nil {
		panic(err)
	}

}
