package services

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/mailru/easyjson"

	"github.com/ASRafalsky/internal/models"
	"github.com/ASRafalsky/internal/repository"
	"github.com/ASRafalsky/internal/server/middleware"
)

func ProcessNewOrder(ctx context.Context, repo orderRepo, buf []byte) int {
	id := cleanSpaces(string(buf))
	if !isValidOrderByLuhn(id) {
		return http.StatusUnprocessableEntity
	}

	login, ok := ctx.Value(middleware.ContextKey("userLogin")).(string)
	if !ok {
		return http.StatusUnauthorized
	}
	userID, err := models.IDHash(login)
	if err != nil {
		return http.StatusInternalServerError
	}
	orderID, err := models.IDHash(id)
	if err != nil {
		return http.StatusInternalServerError
	}

	key, entry, err := repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return http.StatusInternalServerError
	}
	if len(entry) > 0 {
		if !strings.HasPrefix(key, userID) {
			return http.StatusConflict
		}
		return http.StatusOK
	}

	rawOrder := models.Order{
		ID:     id,
		TS:     time.Now(),
		Status: "NEW",
	}

	order, err := easyjson.Marshal(rawOrder)
	if err != nil {
		return http.StatusInternalServerError
	}
	if err := repo.SetOrder(ctx, userID+orderID, order); err != nil {
		return http.StatusInternalServerError
	}
	repo.PushOrder(repository.OrderInfo{
		User:  userID,
		Order: rawOrder,
	})
	return http.StatusAccepted
}

func ListOrders(ctx context.Context, repo orderRepo) ([]byte, int) {
	login, ok := ctx.Value(middleware.ContextKey("userLogin")).(string)
	if !ok {
		return nil, http.StatusUnauthorized
	}
	userID, err := models.IDHash(login)
	if err != nil {
		return nil, http.StatusInternalServerError
	}

	_, entries, err := repo.GetOrdersByUser(ctx, userID)
	if err != nil {
		return nil, http.StatusInternalServerError
	}
	if len(entries) == 0 {
		return nil, http.StatusNoContent
	}

	orders := make([]models.Order, 0, len(entries))
	for i := range entries {
		var order models.Order
		if err = easyjson.Unmarshal(entries[i], &order); err != nil {
			return nil, http.StatusInternalServerError
		}
		if len(order.Status) > 0 && order.Status != "REGISTERED" {
			orders = append(orders, order)
		}
	}

	dataToSend, err := json.Marshal(orders)
	if err != nil {
		return nil, http.StatusInternalServerError
	}

	return dataToSend, http.StatusOK
}

type orderRepo interface {
	SetOrder(ctx context.Context, key string, data []byte) error
	GetOrderByID(ctx context.Context, key string) (string, []byte, error)
	GetOrdersByUser(ctx context.Context, key string) ([]string, [][]byte, error)
	PushOrder(order repository.OrderInfo)
}
