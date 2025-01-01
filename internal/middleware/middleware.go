package middleware

import (
	"context"
	"net/http"

	"github.com/zyrterviews/chirpy/internal/appenv"
	"github.com/zyrterviews/chirpy/internal/auth"
)

type Middleware func(env *appenv.Env) func(next http.Handler) http.Handler

func Chain(env *appenv.Env, middlewares ...Middleware) http.Handler {
	var handler http.Handler
	for i := range middlewares {
		// FIFO
		handler = middlewares[len(middlewares)-1-i](env)(handler)
	}

	return handler
}

func New(handler http.Handler) Middleware {
	return func(_ *appenv.Env) func(next http.Handler) http.Handler {
		return func(_ http.Handler) http.Handler {
			return http.HandlerFunc(
				func(writer http.ResponseWriter, req *http.Request) {
					handler.ServeHTTP(writer, req)
				},
			)
		}
	}
}

func Authenticate(env *appenv.Env) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(writer http.ResponseWriter, req *http.Request) {
				token, err := auth.GetBearerToken(req.Header)
				if err != nil {
					http.Error(writer, err.Error(), http.StatusUnauthorized)

					return
				}

				userID, err := auth.ValidateJWT(token, env.JWTSecret)
				if err != nil {
					http.Error(writer, err.Error(), http.StatusUnauthorized)

					return
				}

				env.UserID = userID

				next.ServeHTTP(writer, req)
			},
		)
	}
}

func WithPrivileges(
	privileges ...func(ctx context.Context, env *appenv.Env) (bool, *auth.AuthError),
) Middleware {
	return func(env *appenv.Env) func(next http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(
				func(writer http.ResponseWriter, req *http.Request) {
					for _, privilege := range privileges {
						ok, err := privilege(req.Context(), env)
						if err != nil {
							http.Error(writer, err.Error(), err.Status)
						}

						if !ok {
							http.Error(
								writer,
								"FORBIDDEN",
								http.StatusForbidden,
							)

							return
						}
					}

					next.ServeHTTP(writer, req)
				},
			)
		}
	}
}

func MetricsInc(env *appenv.Env) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(writer http.ResponseWriter, req *http.Request) {
				env.FileserverHits.Add(1)
				next.ServeHTTP(writer, req)
			},
		)
	}
}
