package endpoints

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleRoot handles the root endpoint.
func HandleRoot(w http.ResponseWriter, r *http.Request) {
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
