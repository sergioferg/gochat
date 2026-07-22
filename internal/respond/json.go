package respond

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

func WithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		logrus.Warn(err)
	}
	if code > 499 {
		logrus.Warn("Responding with 5XX error:", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	WithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func WithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(payload)
	if err != nil {
		logrus.Warn("Error marshalling json:", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(data)
}
