package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/zyrterviews/chirpy/internal/appenv"
)

// GET /admin/metrics
func GetAdminMetrics(env *appenv.Env) http.Handler {
	return http.StripPrefix("/admin/",
		http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")

			fmt.Fprintf(writer, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, env.FileserverHits.Load())
		}),
	)
}

func PostAdminReset(env *appenv.Env) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, req *http.Request) {
			if os.Getenv("PLATFORM") != "dev" {
				writer.WriteHeader(http.StatusForbidden)

				return
			}

			if err := env.DB.DeleteAllUsers(req.Context()); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)

				return
			}

			env.FileserverHits.Swap(0)

			writer.WriteHeader(http.StatusOK)
		},
	)
}
