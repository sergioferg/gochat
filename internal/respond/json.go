package respond

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/sergioferg/gochat/internal/apperrors"
	"github.com/sirupsen/logrus"
)

func RespondWithError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		if appErr.Err != nil {
			logrus.Warnf("AppError %d: %s (internal: %v)", appErr.Code, appErr.Message, appErr.Err)
		}
		WithJSON(w, appErr.Code, map[string]string{"error": appErr.Message})
		return
	}

	logrus.Errorf("Internal Server Error: %v", err)
	WithJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal Server Error"})
}

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
