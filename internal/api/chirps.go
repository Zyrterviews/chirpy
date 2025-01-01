package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/zyrterviews/chirpy/internal/appenv"
	"github.com/zyrterviews/chirpy/internal/database"
)

const maxChirpLength int = 140

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

// POST /api/chirps
func PostOneChirp(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
			if env.UserID == uuid.Nil {
				http.Error(writer, "UNAUTHORIZED", http.StatusUnauthorized)

				return
			}

			type input struct {
				Body string `json:"body"`
			}

			profanities := []string{
				"kerfuffle",
				"sharbert",
				"fornax",
			}

			writer.Header().Set("Content-Type", "application/json")

			var data input

			decoder := json.NewDecoder(req.Body)

			if err := decoder.Decode(&data); err != nil {
				resData := struct {
					Error string `json:"error"`
				}{Error: "Something went wrong"}
				res, _ := json.Marshal(resData)

				writer.WriteHeader(http.StatusInternalServerError)
				_, _ = writer.Write(res)

				return
			}

			if len(data.Body) > maxChirpLength {
				resData := struct {
					Error string `json:"error"`
				}{Error: "Chirp is too long"}
				res, _ := json.Marshal(resData)

				writer.WriteHeader(http.StatusBadRequest)
				_, _ = writer.Write(res)

				return
			}

			for _, p := range profanities {
				re := regexp.MustCompile("(?i)" + p)
				data.Body = re.ReplaceAllString(data.Body, "****")
			}

			opts := database.CreateChirpParams{
				Body:   data.Body,
				UserID: env.UserID,
			}

			chirp, err := env.DB.CreateChirp(req.Context(), opts)
			if err != nil {
				resData := struct {
					Error string `json:"error"`
				}{Error: "Something went wrong"}
				res, _ := json.Marshal(resData)

				writer.WriteHeader(http.StatusInternalServerError)
				_, _ = writer.Write(res)

				return
			}

			resData := Chirp{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			}

			res, err := json.Marshal(resData)
			if err != nil {
				resData := struct {
					Error string `json:"error"`
				}{Error: "Something went wrong"}
				res, _ := json.Marshal(resData)

				writer.WriteHeader(http.StatusInternalServerError)
				_, _ = writer.Write(res)
			}

			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write(res)
		},
	)
}

// GET /api/chirps/
func GetAllChirps(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
			authorID := req.URL.Query().Get("author_id")
			sort := req.URL.Query().Get("sort")

			var (
				chirps []database.Chirp
				err    error
			)

			switch authorID {
			case "":
				chirps, err = env.DB.GetAllChirps(req.Context(), sort)
				if err != nil {
					http.Error(
						writer,
						err.Error(),
						http.StatusInternalServerError,
					)

					return
				}
			default:
				id, perr := uuid.Parse(authorID)
				if perr != nil {
					http.Error(
						writer,
						perr.Error(),
						http.StatusInternalServerError,
					)
				}

				opts := database.GetAllChirpsForUserParams{
					UserID:  id,
					Column2: sort,
				}

				chirps, err = env.DB.GetAllChirpsForUser(req.Context(), opts)
				if err != nil {
					http.Error(
						writer,
						err.Error(),
						http.StatusInternalServerError,
					)

					return
				}
			}

			resData := make([]Chirp, 0, len(chirps))

			for _, chirp := range chirps {
				resData = append(resData, Chirp{
					ID:        chirp.ID,
					CreatedAt: chirp.CreatedAt,
					UpdatedAt: chirp.UpdatedAt,
					Body:      chirp.Body,
					UserID:    chirp.UserID,
				})
			}

			res, err := json.Marshal(&resData)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			_, _ = writer.Write(res)
		},
	)
}

// GET /api/chirps/{chirpID}
func GetOneChirpByID(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
			id, err := uuid.Parse(req.PathValue("chirpID"))
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			chirp, err := env.DB.GetChirpByID(req.Context(), id)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					http.NotFound(writer, req)

					return
				}

				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			resData := Chirp{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			}

			res, err := json.Marshal(&resData)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			_, _ = writer.Write(res)
		},
	)
}

// DELETE /api/chirps/{chirpID}
func DeleteChirpByID(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
			if env.UserID == uuid.Nil {
				http.Error(writer, "UNAUTHORIZED", http.StatusUnauthorized)

				return
			}

			id, err := uuid.Parse(req.PathValue("chirpID"))
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			chirp, err := env.DB.GetChirpByID(req.Context(), id)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			if chirp.UserID != env.UserID {
				http.Error(writer, "FORBIDDEN", http.StatusForbidden)

				return
			}

			err = env.DB.DeleteChirpByID(req.Context(), id)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			writer.WriteHeader(http.StatusNoContent)
		},
	)
}
