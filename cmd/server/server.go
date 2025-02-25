package main

import (
	"context"
	"encoding/json"
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
	"github.com/hoyle1974/khronoscope/internal/metrics"
	"github.com/hoyle1974/khronoscope/internal/resources"
)

var d dao.KhronoStore

func main() {
	// Init logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Init config
	cfg, err := config.InitConfig()
	if err != nil {
		log.Panic().Err(err).Msg("problem initializing config")
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
	d = dao.New()

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
		log.Panic().Err(err).Msg("watch failed")
	}

	// Register the request handler.
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/resources", handleResources)
	http.HandleFunc("/query/range", handleQueryRange)
	http.HandleFunc("/", handleRoot)

	// Start the HTTP server.
	port := 8080
	log.Info().Int("Port", port).Msg("Server listening")
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Error().Err(err).Msg("error starting server:")
	}
}

// Data represents the structure of the JSON response.
type Data struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// handleRequest handles incoming HTTP requests.
func handleResources(w http.ResponseWriter, r *http.Request) {
	// Set the Content-Type header to application/json.
	w.Header().Set("Content-Type", "application/json")

	timestampStr := r.URL.Query().Get("timestamp")

	var timestamp time.Time
	if timestampStr != "now" {
		var err error
		timestamp, err = time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		timestamp = time.Now()
	}

	resources := d.GetResourcesAt(timestamp, "", "")

	// Encode the data struct to JSON.
	jsonData, err := json.Marshal(resources)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the JSON response.
	if _, err := w.Write(jsonData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleHealth handles the /health endpoint.
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// TimestampRange represents min and max timestamps for /query/range.
type TimestampRange struct {
	MinTimestamp time.Time `json:"minTimestamp"`
	MaxTimestamp time.Time `json:"maxTimestamp"`
}

// handleQueryRange handles the /query/range endpoint.
func handleQueryRange(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	minTimestamp, maxTimestamp := d.GetTimeRange()

	timestampRange := TimestampRange{
		MinTimestamp: minTimestamp,
		MaxTimestamp: maxTimestamp,
	}

	jsonData, err := json.Marshal(timestampRange)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(jsonData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleRoot handles the root endpoint.
func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	data := Data{
		Message:   "Hello from Go!",
		Timestamp: time.Now().UTC(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(jsonData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
