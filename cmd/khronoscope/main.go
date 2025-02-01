package main

import (
	"encoding/gob"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/dao"
	khronoscope "github.com/hoyle1974/khronoscope/internal/program"
	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/ui"
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

	d := dao.New()

	if len(filename) > 0 {
		d = dao.NewFromFile(filename)
	}
	var watcher = resources.NewK8sWatcher(d)
	var logCollector = resources.NewLogCollector(client)

	if len(filename) > 0 {
		watcher = nil
	}

	err = watcher.Watch(client, d)
	if err != nil {
		panic(err)
	}

	appModel := khronoscope.NewProgram(watcher, d, logCollector)
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
