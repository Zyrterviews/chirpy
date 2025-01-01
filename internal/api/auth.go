//nolint:exhaustruct
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/zyrterviews/chirpy/internal/appenv"
	"github.com/zyrterviews/chirpy/internal/auth"
	"github.com/zyrterviews/chirpy/internal/database"
)

const (
	jwtExpirationTime          = 1 * time.Hour
	refreshTokenExpirationTime = 60 * 24 * time.Hour // 60 days
)

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
}

// POST /api/login
func Login(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
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

			user, err := env.DB.GetUserByEmail(req.Context(), data.Email)
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

			token, err := auth.MakeJWT(user.ID, env.JWTSecret, jwtExpiresIn)
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

			_, err = env.DB.CreateRefreshToken(req.Context(), opts)
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
				IsChirpyRed:  user.IsChirpyRed,
			}

			res, err := json.Marshal(&resData)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			writer.Header().Set("Content-Type", "application/json")

			_, _ = writer.Write(res)
		},
	)
}

// POST /api/refresh
func Refresh(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
			token, err := auth.GetBearerToken(req.Header)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)

				return
			}

			if dbToken, err := env.DB.GetRefreshToken(req.Context(), token); err != nil ||
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

			user, err := env.DB.GetUserFromRefreshToken(req.Context(), token)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			newToken, err := auth.MakeJWT(
				user.ID,
				env.JWTSecret,
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
		},
	)
}

// POST /api/revoke
func Revoke(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
			token, err := auth.GetBearerToken(req.Header)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)

				return
			}

			err = env.DB.RevokeRefreshToken(req.Context(), token)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)

				return
			}

			writer.WriteHeader(http.StatusNoContent)
		},
	)
}

// POST /api/users
func Signup(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
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

			user, err := env.DB.CreateUser(req.Context(), opts)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			resData := User{
				ID:          user.ID,
				CreatedAt:   user.CreatedAt,
				UpdatedAt:   user.UpdatedAt,
				Email:       user.Email,
				IsChirpyRed: user.IsChirpyRed,
			}

			res, err := json.Marshal(&resData)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			writer.WriteHeader(http.StatusCreated)
			writer.Header().Set("Content-Type", "application/json")

			_, _ = writer.Write(res)
		},
	)
}

// PUT /api/users
func PutUser(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
			if env.UserID == uuid.Nil {
				http.Error(writer, "UNAUTHORIZED", http.StatusUnauthorized)

				return
			}

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

			opts := database.UpdateUserParams{
				ID:             env.UserID,
				Email:          data.Email,
				HashedPassword: hashedPwd,
			}

			newUser, err := env.DB.UpdateUser(req.Context(), opts)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			resData := User{
				ID:          newUser.ID,
				CreatedAt:   newUser.CreatedAt,
				UpdatedAt:   newUser.UpdatedAt,
				Email:       newUser.Email,
				IsChirpyRed: newUser.IsChirpyRed,
			}

			res, err := json.Marshal(resData)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			_, _ = writer.Write(res)
		},
	)
}
