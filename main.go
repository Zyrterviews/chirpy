//nolint:exhaustruct
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/zyrterviews/chirpy/app"
	"github.com/zyrterviews/chirpy/internal/api"
	"github.com/zyrterviews/chirpy/internal/appenv"
	"github.com/zyrterviews/chirpy/internal/database"
	"github.com/zyrterviews/chirpy/internal/middleware"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	env := &appenv.Env{
		DB:             database.New(db),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		FileserverHits: &atomic.Int32{},
	}

	mux := http.NewServeMux()

	// APP
	mux.Handle("/app/", middleware.MetricsInc(env)(app.GetStaticAssets()))

	// API
	mux.Handle("GET /api/healthz", api.GetHealthz())

	mux.Handle(
		"POST /api/chirps",
		middleware.Chain(
			env,
			middleware.Authenticate,
			middleware.New(api.PostOneChirp(env)),
		),
	)

	mux.Handle("DELETE /api/chirps/{chirpID}",
		middleware.Chain(env,
			middleware.Authenticate,
			middleware.New(api.DeleteChirpByID(env)),
		),
	)

	mux.Handle("GET /api/chirps", api.GetAllChirps(env))
	mux.Handle("GET /api/chirps/{chirpID}", api.GetOneChirpByID(env))

	mux.Handle("POST /api/login", api.Login(env))
	mux.Handle("POST /api/refresh", api.Refresh(env))
	mux.Handle("POST /api/revoke", api.Revoke(env))
	mux.Handle("POST /api/users", api.Signup(env))

	mux.Handle(
		"PUT /api/users",
		middleware.Chain(
			env,
			middleware.Authenticate,
			middleware.New(api.PutUser(env)),
		),
	)

	mux.Handle("POST /api/polka/webhooks", api.PostPolkaUpradeUser(env))

	// ADMIN
	mux.Handle("GET /admin/metrics", api.GetAdminMetrics(env))
	mux.Handle("POST /admin/reset", api.PostAdminReset(env))

	server := http.Server{
		Handler: mux,
		Addr:    ":8080",
	}

	_ = server.ListenAndServe()
}
