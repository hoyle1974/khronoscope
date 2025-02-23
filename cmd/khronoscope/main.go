package main

import (
	"context"
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

	// Init logging
	ringBuffer := misc.NewRingBuffer(100) // Store last 100 log messages
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: ringBuffer})
	defer func() {
		fmt.Println(ringBuffer.String()) // Output log on exit
	}()

	// Init config
	cfg, err := config.InitConfig()
	if err != nil {
		log.Panic().Err(err).Msg("problem initializing config")
	}

	// Get flags
	filename := flag.StringP("file", "f", "", "Filename to load")
	namespace := flag.StringP("namespace", "n", "", "Namespace to filter on")
	showKeybindings := flag.BoolP("keybindings", "k", false, "Show keybindings")
	kubeConfigFlag := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

	// Show keybindings and then exit
	if *showKeybindings {
		config.Get().KeyBindings.Print()
		return
	}

	// Are we collecting metrics?
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

	// Are we profiling?
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

	// Connect to k8s
	client, err := conn.NewKhronosConnection(kubeConfigFlag)
	if err != nil {
		log.Panic().Err(err).Msg("could not create khronos connection")
	}

	// Create a new data and if we need to replace it from one from a file
	d := dao.New()
	if filename != nil && len(*filename) > 0 {
		d = dao.NewFromFile(*filename)
	}

	// Start the k8s resource watcher
	var watcher = resources.GetK8sWatcher(d)

	// This tool helps us collect logs
	var logCollector = resources.GetLogCollector(client)

	// Stop the watcher and log collector if we are just playing back from a file
	if filename != nil && len(*filename) > 0 {
		watcher = nil
		logCollector = nil
	}

	// Context to be used by the watcher
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching everything
	err = watcher.StartWatching(ctx, client, d, logCollector, *namespace)
	if err != nil {
		log.Panic().Err(err).Msg("watch failed")
	}

	// Start the program
	appModel := khronoscope.NewProgram(watcher, d, logCollector, client, ringBuffer)
	p := tea.NewProgram(appModel)
	appModel.Program = p

	appModel.VCR = ui.NewTimeController(d, func() {
		p.Send(1)
	})

	// If we loaded from a file then pause and set the time to the beginning
	if filename != nil && len(*filename) > 0 {
		min, _ := d.GetTimeRange()
		appModel.VCR.EnableVirtualTime()
		appModel.VCR.SetTime(min)
	}

	// Anytime the watcher sees a change then tell the program to update the display
	watcher.OnChange(func() {
		p.Send(1)
	})

	// Run the program
	if _, err := p.Run(); err != nil {
		log.Panic().Err(err).Msg("program exited")
	}

	// Try to cleanup, but this doesn't always work
	ui.ResetTerminal()
}
