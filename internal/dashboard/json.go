package dashboard

import (
	"encoding/json"
	"io"
	"net/http"
)

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func readJSON(r *http.Request, v interface{}) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		return false
	}
	return json.Unmarshal(body, v) == nil
}