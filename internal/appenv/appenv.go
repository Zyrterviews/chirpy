package appenv

import (
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/zyrterviews/chirpy/internal/database"
)

type Env struct {
	DB             *database.Queries
	JWTSecret      string
	FileserverHits *atomic.Int32
	UserID         uuid.UUID
}
