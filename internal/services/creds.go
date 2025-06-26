package services

import (
	"context"
	"io"
	"net/http"

	"github.com/mailru/easyjson"
	"golang.org/x/crypto/bcrypt"

	"github.com/ASRafalsky/internal/models"
)

func ExtractCreds(r io.Reader) (models.Credential, int) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return models.Credential{}, http.StatusBadRequest
	}

	cred := models.Credential{}
	err = easyjson.Unmarshal(buf, &cred)
	if err != nil {
		return models.Credential{}, http.StatusInternalServerError
	}

	return cred, http.StatusOK
}

func SetUserCred(ctx context.Context, repo credRepo, cred *models.Credential) int {
	if err := models.CheckCreds(cred); err != nil {
		return http.StatusBadRequest
	}

	id, err := models.IDHash(cred.Login)
	if err != nil {
		return http.StatusInternalServerError
	}

	entry, err := repo.GetCred(ctx, id)
	if err != nil {
		return http.StatusInternalServerError
	}

	if len(entry) > 0 {
		return http.StatusConflict
	}

	hashPassword, err := bcrypt.GenerateFromPassword([]byte(cred.Password), bcrypt.DefaultCost)
	if err != nil {
		return http.StatusInternalServerError
	}

	if err := repo.SetCred(ctx, id, hashPassword); err != nil {
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func UserAuth(ctx context.Context, repo credRepo, cred *models.Credential) int {
	if err := models.CheckCreds(cred); err != nil {
		return http.StatusBadRequest
	}

	id, err := models.IDHash(cred.Login)
	if err != nil {
		return http.StatusInternalServerError
	}

	storedPasswdHash, err := repo.GetCred(ctx, id)
	if err != nil {
		return http.StatusInternalServerError
	}

	if err := bcrypt.CompareHashAndPassword(storedPasswdHash, []byte(cred.Password)); err != nil {
		return http.StatusUnauthorized
	}

	return http.StatusOK
}

type credRepo interface {
	SetCred(ctx context.Context, key string, data []byte) error
	GetCred(ctx context.Context, key string) ([]byte, error)
}
