package auth

import (
	"context"
	"fmt"

	"github.com/zyrterviews/chirpy/internal/appenv"
)

type Privilege func(context.Context, *appenv.Env) (bool, *AuthError)

type AuthError struct {
	Err    error
	Status int
}

func (e AuthError) Error() string {
	return fmt.Sprintf("%d %s", e.Status, e.Err)
}

// func CanDeleteChirps(ctx context.Context, env *appenv.Env) (bool, *AuthError) {
// 	if env.UserID == uuid.Nil {
// 		//nolint:exhaustruct
// 		return false, &AuthError{Status: http.StatusUnauthorized}
// 	}

// 	chirp, err := env.DB.GetAllChirpsForUser(ctx, env.UserID)
// 	if err != nil {
// 		return false, &AuthError{
// 			Status: http.StatusInternalServerError,
// 			Err:    err,
// 		}
// 	}

//     // check if chirp is the user's

//     return ?
// }
