package endpoints

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

var startTime = time.Now()

// Healthz endpoint: Simply checks if the application is alive
func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    time.Since(startTime).String(),
	}

	// Write the response as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling healthz response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Write the JSON response.
	if _, err := w.Write(jsonResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
