package endpoints

import (
	"encoding/json"
	"net/http"

	"github.com/hoyle1974/khronoscope/internal/dao"
)

// handleQueryRange handles the /query/range endpoint.
func HandleQueryRange(d dao.KhronoStore, w http.ResponseWriter, r *http.Request) {
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
