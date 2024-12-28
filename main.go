//nolint:err113,forbidigo,exhaustruct,wrapcheck
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/zyrterviews/chirpy/internal/database"
)

const maxChirpLength int = 140

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
			cfg.fileserverHits.Add(1)
			next.ServeHTTP(writer, req)
		},
	)
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	apiCfg := apiConfig{DB: database.New(db)}

	mux := http.NewServeMux()

	mux.Handle(
		"/app/",
		apiCfg.middlewareMetricsInc(
			http.StripPrefix("/app/", http.FileServer(http.Dir("."))),
		),
	)

	// API
	mux.Handle(
		"GET /api/healthz",
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("OK"))
		}),
	)

	mux.Handle(
		"POST /api/chirps",
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			type input struct {
				Body   string    `json:"body"`
				UserID uuid.UUID `json:"user_id"`
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
				UserID: data.UserID,
			}

			chirp, err := apiCfg.DB.CreateChirp(req.Context(), opts)
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
		}),
	)

	mux.Handle(
		"GET /api/chirps",
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			chirps, err := apiCfg.DB.GetAllChirps(req.Context())
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
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
		}),
	)

	mux.Handle(
		"GET /api/chirps/{chirpID}",
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			id, err := uuid.Parse(req.PathValue("chirpID"))
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			chirp, err := apiCfg.DB.GetChirp(req.Context(), id)
			if err != nil {
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
		}),
	)

	mux.Handle(
		"POST /api/users",
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			type input struct {
				Email string `json:"email"`
			}

			var data input

			decoder := json.NewDecoder(req.Body)

			if err := decoder.Decode(&data); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			user, err := apiCfg.DB.CreateUser(req.Context(), data.Email)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			resData := User{
				ID:        user.ID,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
				Email:     user.Email,
			}

			res, err := json.Marshal(&resData)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusCreated)

			_, _ = writer.Write(res)
		}),
	)

	// ADMIN
	mux.Handle(
		"GET /admin/metrics",
		http.StripPrefix("/admin/",
			http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "text/html; charset=utf-8")

				fmt.Fprintf(writer, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, apiCfg.fileserverHits.Load())
			}),
		),
	)

	mux.Handle(
		"POST /admin/reset",
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			if os.Getenv("PLATFORM") != "dev" {
				writer.WriteHeader(http.StatusForbidden)

				return
			}

			if err := apiCfg.DB.DeleteAllUsers(req.Context()); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			apiCfg.fileserverHits.Swap(0)

			writer.WriteHeader(http.StatusOK)
		}),
	)

	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	_ = server.ListenAndServe()
}
