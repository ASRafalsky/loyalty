package services

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/mailru/easyjson"

	"github.com/ASRafalsky/internal/models"
	"github.com/ASRafalsky/internal/server/middleware"
)

func NewWithdraw(ctx context.Context, r io.Reader, repo withdrawalsRepo) int {
	login, ok := ctx.Value(middleware.ContextKey("userLogin")).(string)
	if !ok {
		return http.StatusUnauthorized
	}
	userID, err := models.IDHash(login)
	if err != nil {
		return http.StatusInternalServerError
	}

	buf, err := io.ReadAll(r)
	if err != nil {
		return http.StatusBadRequest
	}

	withdraw := models.InputWithdraw{}
	err = easyjson.Unmarshal(buf, &withdraw)
	if err != nil {
		return http.StatusInternalServerError
	}

	id := cleanSpaces(withdraw.ID)
	if !isValidOrderByLuhn(id) {
		return http.StatusUnprocessableEntity
	}

	entry, err := repo.GetUserStat(ctx, userID)
	if err != nil {
		return http.StatusInternalServerError
	}

	var userStat models.UserStats
	if len(entry) > 0 {
		if err = easyjson.Unmarshal(entry, &userStat); err != nil {
			return http.StatusInternalServerError
		}
	}

	if userStat.CurrentPoints == nil ||
		(userStat.CurrentPoints != nil && *userStat.CurrentPoints < withdraw.Points) {
		return http.StatusPaymentRequired
	} else {
		*userStat.CurrentPoints -= withdraw.Points
	}

	withdrawID, err := models.IDHash(id)
	if err != nil {
		return http.StatusInternalServerError
	}
	withdrawStat := models.Withdraw{
		ID:     id,
		TS:     time.Now(),
		Points: withdraw.Points,
	}
	buf, err = easyjson.Marshal(withdrawStat)
	if err != nil {
		return http.StatusInternalServerError
	}

	if err := repo.SetWithdraw(ctx, userID+withdrawID, buf); err != nil {
		return http.StatusInternalServerError
	}

	if userStat.WithdrawPoints != nil {
		*userStat.WithdrawPoints += withdraw.Points
	} else {
		points := withdraw.Points
		userStat.WithdrawPoints = &points
	}

	buf, err = easyjson.Marshal(userStat)
	if err != nil {
		return http.StatusInternalServerError
	}
	if err := repo.SetUserStat(ctx, userID, buf); err != nil {
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

func GetWithdrawals(ctx context.Context, repo withdrawalsRepo) ([]byte, int) {
	login, ok := ctx.Value(middleware.ContextKey("userLogin")).(string)
	if !ok {
		return nil, http.StatusUnauthorized
	}
	userID, err := models.IDHash(login)
	if err != nil {
		return nil, http.StatusInternalServerError
	}

	_, entries, err := repo.GetWithdrawalsByUser(ctx, userID)
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	if len(entries) == 0 {
		return nil, http.StatusNoContent
	}

	dataToSend := bytes.Join(entries, []byte(","))
	dataToSend = append([]byte("["), dataToSend...)
	dataToSend = append(dataToSend, []byte("]")[0])

	return dataToSend, http.StatusOK
}

type withdrawalsRepo interface {
	SetUserStat(ctx context.Context, key string, data []byte) error
	SetWithdraw(ctx context.Context, key string, data []byte) error
	GetUserStat(ctx context.Context, key string) ([]byte, error)
	GetWithdrawalsByUser(ctx context.Context, key string) ([]string, [][]byte, error)
}
