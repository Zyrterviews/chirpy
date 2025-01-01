package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/zyrterviews/chirpy/internal/appenv"
	"github.com/zyrterviews/chirpy/internal/auth"
)

// POST /api/polka/webhooks
func PostPolkaUpradeUser(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
			apiKey, err := auth.GetAPIKey(req.Header)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)

				return
			}

			if apiKey != os.Getenv("POLKA_KEY") {
				http.Error(writer, err.Error(), http.StatusUnauthorized)

				return
			}

			type input struct {
				Event string `json:"event"`
				Data  struct {
					UserID string `json:"user_id"`
				} `json:"data"`
			}

			var data input

			decoder := json.NewDecoder(req.Body)

			if err := decoder.Decode(&data); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			if data.Data.UserID == "" {
				http.Error(writer, "Missing user ID", http.StatusBadRequest)

				return
			}

			if data.Event != "user.upgraded" {
				writer.WriteHeader(http.StatusNoContent)

				return
			}

			id, err := uuid.Parse(data.Data.UserID)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			if _, err := env.DB.SetUserAsChirpyRed(req.Context(), id); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					http.NotFound(writer, req)

					return
				}

				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			writer.WriteHeader(http.StatusNoContent)
		},
	)
}
