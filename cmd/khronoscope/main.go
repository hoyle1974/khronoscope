package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hoyle1974/khronoscope/internal/config"
	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/dao"
	"github.com/hoyle1974/khronoscope/internal/metrics"
	"github.com/hoyle1974/khronoscope/internal/misc"
	khronoscope "github.com/hoyle1974/khronoscope/internal/program"
	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/khronoscope/internal/ui"
)

func main() {
	ringBuffer := misc.NewRingBuffer(100) // Store last 5 log messages
	log.SetOutput(ringBuffer)

	cfg, err := config.InitConfig()
	if err != nil {
		panic(err)
	}
	log.Println(cfg)

	done := make(chan bool)
	defer func() {
		done <- true
	}()
	if cfg.Metrics {
		defer metrics.Print()

		ticker := time.NewTicker(5 * time.Second)
		defer func() {
			ticker.Stop()
		}()
		go func() {
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					metrics.Log()
				}
			}
		}()
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

	appModel := khronoscope.NewProgram(watcher, d, logCollector, client, sel, ringBuffer)
	p := tea.NewProgram(appModel)
	appModel.Program = p

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
