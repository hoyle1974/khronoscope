package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"runtime/pprof"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	ringBuffer := misc.NewRingBuffer(100) // Store last 100 log messages
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: ringBuffer})
	defer func() {
		fmt.Println(ringBuffer.String()) // Output log on exit
	}()

	cfg, err := config.InitConfig()
	if err != nil {
		panic(err)
	}

	filename := flag.StringP("file", "f", "", "Filename to load")
	namespace := flag.StringP("namespace", "n", "", "Namespace to filter on")
	showKeybindings := flag.BoolP("keybindings", "k", false, "Show keybindings")
	kubeConfigFlag := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	if *showKeybindings {
		config.Get().KeyBindings.Print()
		return
	}

	if cfg.Metrics {
		done := make(chan bool)
		defer func() {
			done <- true
		}()
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
			log.Panic().Err(err).Msg("error creating khronoscope.pprof file")
		} else {
			if err := pprof.StartCPUProfile(f); err != nil {
				log.Panic().Err(err).Msg("error starting CPU profile")
			}
			defer pprof.StopCPUProfile()
		}
	}

	gob.Register(resources.Resource{})
	gob.Register(resources.NodeExtra{})
	gob.Register(resources.PodExtra{})

	client, err := conn.NewKhronosConnection(kubeConfigFlag)
	if err != nil {
		log.Panic().Err(err).Msg("could not create khronos connection")
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = watcher.Watch(ctx, client, d, logCollector, *namespace)
	if err != nil {
		log.Panic().Err(err).Msg("watch failed")
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
		log.Panic().Err(err).Msg("program exited")
	}

	ui.ResetTerminal()
}
