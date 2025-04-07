package endpoints

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Simulate checking the health of the database or other services
func checkDependencies() map[string]string {
	dependencies := make(map[string]string)

	/*
		// Simulate a database check
		dbStatus := "ok"
		if rand.Float32() < 0.1 { // Simulating a 10% failure rate
			dbStatus = "unreachable"
		}
		dependencies["database"] = dbStatus

		// Simulate a check for an external service
		apiStatus := "ok"
		if rand.Float32() < 0.1 { // Simulating a 10% failure rate
			apiStatus = "down"
		}
		dependencies["external-api"] = apiStatus
	*/

	return dependencies
}

// Readiness endpoint: Checks if all necessary dependencies are up and running
func HandleReadiness(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ok", // Default status
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Check the dependencies (e.g., DB, external services)
	dependencies := checkDependencies()
	response.Dependencies = dependencies

	// If any dependency is "unreachable" or "down", mark the status as "error"
	for _, status := range dependencies {
		if status != "ok" {
			response.Status = "error"
			break
		}
	}

	// Write the response as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling readiness response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Write the JSON response.
	if _, err := w.Write(jsonResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
