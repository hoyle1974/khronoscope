package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"runtime/pprof"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hoyle1974/khronoscope/internal/config"
	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/dao"
	"github.com/hoyle1974/khronoscope/internal/metrics"
	khronoscope "github.com/hoyle1974/khronoscope/internal/program"
	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/ui"
)

func main() {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		log.Println("Recovered from panic:", r)
	// 		fmt.Print("\033[H\033[2J") // Reset the terminal
	// 		os.Exit(1)
	// 	}
	// }()

	////////////////
	// Put the terminal into raw mode to prevent it echoing characters twice.
	// oldState, err := term.MakeRaw(0)
	// if err != nil {
	// 	panic(err)
	// }
	// defer func() {
	// 	err := term.Restore(0, oldState)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }()

	cfg, err := config.InitConfig()
	if err != nil {
		panic(err)
	}
	fmt.Println(cfg)

	if cfg.Metrics {
		defer metrics.Print()
	}

	if cfg.Profiling {
		if f, err := os.Create("khronoscope.pprof"); err != nil {
			log.Fatal(err)
		} else {
			if err := pprof.StartCPUProfile(f); err != nil {
				log.Fatal(err)
			}
			defer pprof.StopCPUProfile()
		}
	}

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
	var watcher = resources.GetK8sWatcher(d)
	var logCollector = resources.GetLogCollector(client)

	if len(filename) > 0 {
		watcher = nil
	}

	err = watcher.Watch(client, d, logCollector)
	if err != nil {
		panic(err)
	}

	// Wait to make sure we connect to something
	var sel resources.Resource
	// time.Sleep(time.Second * 2)
	// for _, r := range d.GetResourcesAt(time.Now(), "Pod", "kube-system") {
	// 	if r.GetName() == "etcd-kind-control-plane" {
	// 		sel = r
	// 		break
	// 	}
	// }

	appModel := khronoscope.NewProgram(watcher, d, logCollector, client, sel)
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

	ui.ResetTerminal()
}
