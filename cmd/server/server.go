package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime/pprof"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	flag "github.com/spf13/pflag"

	"github.com/hoyle1974/khronoscope/internal/config"
	"github.com/hoyle1974/khronoscope/internal/conn"
	"github.com/hoyle1974/khronoscope/internal/dao"
	"github.com/hoyle1974/khronoscope/internal/endpoints"
	"github.com/hoyle1974/khronoscope/internal/metrics"
	"github.com/hoyle1974/khronoscope/internal/misc"
	"github.com/hoyle1974/khronoscope/internal/resources"
	"github.com/hoyle1974/temporal"

	"github.com/hoyle1974/temporal/storage"
)

func logErr(err error, msg string) {
	log.Error().Any("stack", fmt.Sprintf("%+v", err)).Err(err).Msg(msg)
	fmt.Println("---------------------------------")
	fmt.Printf("%+v", err)
}

func main() {

	// Init logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// Init config
	cfg, err := config.InitServerConfig()
	if err != nil {
		log.Error().Err(err).Msg("problem initializing config")
	}

	// Get flags
	namespace := flag.StringP("namespace", "n", "", "Namespace to filter on")
	kubeConfigFlag := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	flag.Parse()

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
			logErr(err, "error creating khronoscope.pprof file")
			return
		} else {
			if err := pprof.StartCPUProfile(f); err != nil {
				logErr(err, "error starting CPU profile")
				return
			}
			defer pprof.StopCPUProfile()
		}
	}

	// Connect to k8s
	client, err := conn.NewKhronosConnection(kubeConfigFlag)
	if err != nil {
		logErr(err, "could not create khronos connection")
		return
	}

	storage := storage.NewDiskStorage("/Users/jstrohm/code/khronoscope/server-data")
	if err != nil {
		logErr(err, "error creating new storage manager")
		return
	}

	temporalMap, err := temporal.NewMapWithConfig(storage, temporal.MapConfig{
		MaxChunkTargetSize: 8 * 1024 * 1024,
		MaxChunkAge:        1 * time.Minute,
		Logger:             misc.ZeroLogTemporalLoggerAdapter{},
		Metrics:            metrics.TemporalMetricsAdapter{},
	})
	if err != nil {
		logErr(err, "error creating new temporal map")
		return
	}

	d, err := dao.NewStore(temporalMap)
	if err != nil {
		logErr(err, "could not create dao")
		return
	}

	// Start the k8s resource watcher
	var watcher = resources.GetK8sWatcher(d)

	// This tool helps us collect logs
	var logCollector = resources.GetLogCollector(client)

	// Context to be used by the watcher
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watching everything
	err = watcher.StartWatching(ctx, client, d, logCollector, *namespace)
	if err != nil {
		logErr(err, "watch failed")
		return
	}

	// Register the request handler.
	http.HandleFunc("/healthz", endpoints.HandleHealthz)
	http.HandleFunc("/readiness", endpoints.HandleReadiness)
	http.HandleFunc("/resources", func(w http.ResponseWriter, r *http.Request) {
		endpoints.HandleResources(d, w, r)
	})
	http.HandleFunc("/query/range", func(w http.ResponseWriter, r *http.Request) {
		endpoints.HandleQueryRange(d, w, r)
	})
	http.HandleFunc("/", endpoints.HandleRoot)

	// Start the HTTP server.
	port := 8080
	log.Info().Int("Port", port).Msg("Server listening")
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		logErr(err, "error starting server:")
		return
	}
}
