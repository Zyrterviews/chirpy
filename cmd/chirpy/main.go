//nolint:exhaustruct
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
	"github.com/zyrterviews/chirpy/internal/auth"
	"github.com/zyrterviews/chirpy/internal/database"
)

const (
	maxChirpLength             int = 140
	jwtExpirationTime              = 1 * time.Hour
	refreshTokenExpirationTime     = 60 * 24 * time.Hour // 60 days
)

type apiConfig struct {
	fileserverHits atomic.Int32
	DB             *database.Queries
	JWTSecret      string
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
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
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

	apiCfg := apiConfig{
		DB:        database.New(db),
		JWTSecret: os.Getenv("JWT_SECRET"),
	}

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

			token, err := auth.GetBearerToken(req.Header)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)

				return
			}

			userID, err := auth.ValidateJWT(token, apiCfg.JWTSecret)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)

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
				UserID: userID,
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

			chirp, err := apiCfg.DB.GetChirpByID(req.Context(), id)
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
		"POST /api/login",
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			type input struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}

			var data input

			decoder := json.NewDecoder(req.Body)

			if err := decoder.Decode(&data); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			user, err := apiCfg.DB.GetUserByEmail(req.Context(), data.Email)
			if err != nil {
				http.Error(
					writer,
					"email or password does not match",
					http.StatusUnauthorized,
				)

				return
			}

			if err := auth.CheckPasswordHash(data.Password, user.HashedPassword); err != nil {
				http.Error(
					writer,
					"email or password does not match",
					http.StatusUnauthorized,
				)

				return
			}

			jwtExpiresIn := jwtExpirationTime

			token, err := auth.MakeJWT(user.ID, apiCfg.JWTSecret, jwtExpiresIn)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			refreshToken, err := auth.MakeRefreshToken()
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			opts := database.CreateRefreshTokenParams{
				Token:     refreshToken,
				UserID:    user.ID,
				ExpiresAt: time.Now().UTC().Add(refreshTokenExpirationTime),
			}

			_, err = apiCfg.DB.CreateRefreshToken(req.Context(), opts)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			resData := User{
				ID:           user.ID,
				CreatedAt:    user.CreatedAt,
				UpdatedAt:    user.UpdatedAt,
				Email:        user.Email,
				Token:        token,
				RefreshToken: refreshToken,
			}

			res, err := json.Marshal(&resData)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			writer.Header().Set("Content-Type", "application/json")

			_, _ = writer.Write(res)
		}),
	)

	mux.Handle(
		"POST /api/refresh",
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			token, err := auth.GetBearerToken(req.Header)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)

				return
			}

			if dbToken, err := apiCfg.DB.GetRefreshToken(req.Context(), token); err != nil ||
				dbToken.ExpiresAt.Before(time.Now().UTC()) ||
				dbToken.ExpiresAt.Equal(time.Now().UTC()) ||
				dbToken.RevokedAt.Valid {
				msg := "token expired"

				if err != nil {
					msg = err.Error()
				}

				http.Error(writer, msg, http.StatusUnauthorized)

				return
			}

			user, err := apiCfg.DB.GetUserFromRefreshToken(req.Context(), token)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			newToken, err := auth.MakeJWT(
				user.ID,
				apiCfg.JWTSecret,
				jwtExpirationTime,
			)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			resData := struct {
				Token string `json:"token"`
			}{Token: newToken}

			res, err := json.Marshal(resData)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			_, _ = writer.Write(res)
		}),
	)

	mux.Handle(
		"POST /api/revoke",
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			token, err := auth.GetBearerToken(req.Header)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)

				return
			}

			err = apiCfg.DB.RevokeRefreshToken(req.Context(), token)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)

				return
			}

			writer.WriteHeader(http.StatusNoContent)
		}),
	)

	mux.Handle(
		"POST /api/users",
		http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			type input struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}

			var data input

			decoder := json.NewDecoder(req.Body)

			if err := decoder.Decode(&data); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			hashedPwd, err := auth.HashPassword(data.Password)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			opts := database.CreateUserParams{
				Email:          data.Email,
				HashedPassword: hashedPwd,
			}

			user, err := apiCfg.DB.CreateUser(req.Context(), opts)
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

			writer.WriteHeader(http.StatusCreated)
			writer.Header().Set("Content-Type", "application/json")

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
