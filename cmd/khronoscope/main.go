package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"

	flag "github.com/spf13/pflag"

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

	//"/Users/jstrohm/code/khronoscope/session.khron"
	filename := flag.StringP("file", "f", "", "Filename to load")
	namespace := flag.StringP("namespace", "n", "", "Namespace to filter on")
	showKeybindings := flag.BoolP("keybindings", "k", false, "Show keybindings")
	kubeConfigFlag := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	if *showKeybindings {
		config.Get().KeyBindings.Print()
		return
	}

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
	gob.Register(resources.NodeExtra{})
	gob.Register(resources.PodExtra{})
	gob.Register(resources.GenericExtra{})

	client, err := conn.NewKhronosConnection(kubeConfigFlag)
	if err != nil {
		fmt.Printf("Error creating connection: %v", err)
		return
	}

	d := dao.New()

	if filename != nil && len(*filename) > 0 {
		d = dao.NewFromFile(*filename)
	}
	var watcher = resources.GetK8sWatcher(d)
	var logCollector = resources.GetLogCollector(client)

	if filename != nil && len(*filename) > 0 {
		watcher = nil
		logCollector = nil
	}

	err = watcher.Watch(client, d, logCollector, *namespace)
	if err != nil {
		panic(err)
	}

	appModel := khronoscope.NewProgram(watcher, d, logCollector, client, ringBuffer)
	p := tea.NewProgram(appModel)
	appModel.Program = p

	appModel.VCR = ui.NewTimeController(d, func() {
		p.Send(1)
	})

	if filename != nil && len(*filename) > 0 {
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
