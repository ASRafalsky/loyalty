package services

import (
	"context"
	"net/http"

	"github.com/ASRafalsky/internal/models"
	"github.com/ASRafalsky/internal/server/middleware"
)

func GetUserBalance(ctx context.Context, repo userRepo) ([]byte, int) {
	login, ok := ctx.Value(middleware.ContextKey("userLogin")).(string)
	if !ok {
		return nil, http.StatusUnauthorized
	}
	userID, err := models.IDHash(login)
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	entry, err := repo.GetUserStat(ctx, userID)
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	return entry, http.StatusOK
}

type userRepo interface {
	SetUserStat(ctx context.Context, key string, data []byte) error
	GetUserStat(ctx context.Context, key string) ([]byte, error)
}
