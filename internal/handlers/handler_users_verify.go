package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/respond"
	"github.com/sirupsen/logrus"
)

func (api *API) HandlerUserVerify(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Token string `json:"token"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respond.WithError(w, http.StatusBadRequest, "Couldn't decode parameters", err)
		return
	}

	if params.Token == "" {
		respond.WithError(w, http.StatusBadRequest, "Missing verification token", nil)
		return
	}

	incomingHash := auth.HashToken(params.Token)
	user, err := api.DB.GetUserFromVerificationToken(r.Context(), incomingHash)
	if err != nil {
		if err == sql.ErrNoRows {
			respond.WithError(w, http.StatusConflict, "Token is invalid or expired", nil)
			return
		}
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	err = api.DB.VerifyUser(r.Context(), user.ID)
	if err != nil {
		respond.WithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	err = api.DB.DeleteVerificationTokensByUserID(r.Context(), user.ID)
	if err != nil {
		logrus.Error("Failed to delete verification token after use:", err)
	}

	type response struct {
		Message string `json:"message"`
	}
	respond.WithJSON(w, http.StatusOK, response{
		Message: "Account successfully verified. You may now log in.",
	})
}
