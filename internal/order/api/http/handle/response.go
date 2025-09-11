package handle

import (
	"encoding/json"
	"net/http"
)

// jsonResponse writes the given data as a JSON-encoded HTTP response with status code 200 OK.
func jsonResponse(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if data == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(data)
}

// jsonError writes an error response as JSON with the specified HTTP status code.
func jsonError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
		"code":  code,
	})
}
