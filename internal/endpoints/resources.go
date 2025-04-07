package endpoints

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/hoyle1974/khronoscope/internal/dao"
)

// handleRequest handles incoming HTTP requests.
func HandleResources(d dao.KhronoStore, w http.ResponseWriter, r *http.Request) {
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
