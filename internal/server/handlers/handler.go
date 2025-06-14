package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mailru/easyjson"
	"golang.org/x/crypto/bcrypt"

	"github.com/ASRafalsky/internal/models"
	"github.com/ASRafalsky/internal/repository"
	"github.com/ASRafalsky/internal/server/authorization"
	"github.com/ASRafalsky/internal/server/middleware"
)

func RegisterPostHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		buf, err := io.ReadAll(req.Body)
		if err != nil {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		defer func() {
			// Handled at the logging level.
			_ = req.Body.Close()
		}()
		cred := models.Credential{}
		err = easyjson.Unmarshal(buf, &cred)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !checkLogin(cred.Login) || !checkPassword(cred.Password) {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		id, err := models.IDHash(cred.Login)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		entry, err := repo.GetCred(req.Context(), id)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(entry) > 0 {
			res.WriteHeader(http.StatusConflict)
			return
		}

		hashPassword, err := bcrypt.GenerateFromPassword([]byte(cred.Password), bcrypt.DefaultCost)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := repo.SetCred(req.Context(), id, hashPassword); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		expirationTime := time.Now().Add(15 * time.Minute)
		token, err := authorization.NewToken(cred.Login, expirationTime)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := setToken(res, req, token, expirationTime); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusOK)
	}
}

func LoginPostHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		buf, err := io.ReadAll(req.Body)
		if err != nil {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		defer func() {
			// Handled at the logging level.
			_ = req.Body.Close()
		}()
		cred := models.Credential{}
		err = easyjson.Unmarshal(buf, &cred)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !checkLogin(cred.Login) || !checkPassword(cred.Password) {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		id, err := models.IDHash(cred.Login)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		storedPasswdHash, err := repo.GetCred(req.Context(), id)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := bcrypt.CompareHashAndPassword(storedPasswdHash, []byte(cred.Password)); err != nil {
			http.Error(res, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		expirationTime := time.Now().Add(15 * time.Minute)
		token, err := authorization.NewToken(cred.Login, expirationTime)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := setToken(res, req, token, expirationTime); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusOK)
	}
}

func OrdersPostHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		buf, err := io.ReadAll(req.Body)
		if err != nil {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		defer func() {
			// Handled at the logging level.
			_ = req.Body.Close()
		}()
		if req.Header.Get("Content-Type") != "text/plain" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		id := cleanSpaces(string(buf))
		if !isValidOrderByLuhn(id) {
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		login, ok := req.Context().Value(middleware.ContextKey("userLogin")).(string)
		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		userID, err := models.IDHash(login)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		orderID, err := models.IDHash(id)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		key, entry, err := repo.GetOrderByID(req.Context(), orderID)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(entry) > 0 {
			if !strings.HasPrefix(key, userID) {
				res.WriteHeader(http.StatusConflict)
				return
			}
			res.WriteHeader(http.StatusOK)
			return
		}

		rawOrder := models.Order{
			ID:     id,
			TS:     time.Now(),
			Status: "NEW",
		}

		order, err := easyjson.Marshal(rawOrder)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := repo.SetOrder(req.Context(), userID+orderID, order); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		repo.PushOrder(repository.OrderInfo{
			User:  userID,
			Order: rawOrder,
		})
		res.WriteHeader(http.StatusAccepted)
	}
}

func OrdersGetHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if len(req.Header.Get("Content-Length")) > 0 && req.Header.Get("Content-Length") != "0" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		login, ok := req.Context().Value(middleware.ContextKey("userLogin")).(string)
		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		userID, err := models.IDHash(login)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, entries, err := repo.GetOrdersByUser(req.Context(), userID)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(entries) == 0 {
			res.WriteHeader(http.StatusNoContent)
			return
		}

		orders := make([]models.Order, 0, len(entries))
		for i := range entries {
			var order models.Order
			if err = easyjson.Unmarshal(entries[i], &order); err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				return
			}
			if len(order.Status) > 0 && order.Status != "REGISTERED" {
				orders = append(orders, order)
			}
		}

		dataToSend, err := json.Marshal(orders)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		if _, err = res.Write(dataToSend); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusOK)
	}
}

func BalanceGetHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if len(req.Header.Get("Content-Length")) > 0 && req.Header.Get("Content-Length") != "0" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		login, ok := req.Context().Value(middleware.ContextKey("userLogin")).(string)
		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		userID, err := models.IDHash(login)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		entry, err := repo.GetUserStat(req.Context(), userID)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		if _, err = res.Write(entry); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusOK)
	}
}

func WithdrawPostHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "application/json" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		login, ok := req.Context().Value(middleware.ContextKey("userLogin")).(string)
		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		userID, err := models.IDHash(login)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		buf, err := io.ReadAll(req.Body)
		if err != nil {
			res.WriteHeader(http.StatusNotFound)
			return
		}
		defer func() {
			// Handled at the logging level.
			_ = req.Body.Close()
		}()

		withdraw := models.InputWithdraw{}
		err = easyjson.Unmarshal(buf, &withdraw)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		id := cleanSpaces(withdraw.ID)
		if !isValidOrderByLuhn(id) {
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		entry, err := repo.GetUserStat(req.Context(), userID)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		var userStat models.UserStats
		if len(entry) > 0 {
			if err = easyjson.Unmarshal(entry, &userStat); err != nil {
				res.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		if userStat.CurrentPoints == nil ||
			(userStat.CurrentPoints != nil && *userStat.CurrentPoints < withdraw.Points) {
			res.WriteHeader(http.StatusPaymentRequired)
			return
		} else {
			*userStat.CurrentPoints -= withdraw.Points
		}

		withdrawID, err := models.IDHash(id)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		withdrawStat := models.Withdraw{
			ID:     id,
			TS:     time.Now(),
			Points: withdraw.Points,
		}
		buf, err = easyjson.Marshal(withdrawStat)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := repo.SetWithdraw(req.Context(), userID+withdrawID, buf); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		if userStat.WithdrawPoints != nil {
			*userStat.WithdrawPoints += withdraw.Points
		} else {
			points := withdraw.Points
			userStat.WithdrawPoints = &points
		}

		buf, err = easyjson.Marshal(userStat)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := repo.SetUserStat(req.Context(), userID, buf); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusOK)
	}
}

func WithdrawalsGetHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if len(req.Header.Get("Content-Length")) > 0 && req.Header.Get("Content-Length") != "0" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		login, ok := req.Context().Value(middleware.ContextKey("userLogin")).(string)
		if !ok {
			res.WriteHeader(http.StatusUnauthorized)
			return
		}
		userID, err := models.IDHash(login)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, entries, err := repo.GetWithdrawalsByUser(req.Context(), userID)
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(entries) == 0 {
			res.WriteHeader(http.StatusNoContent)
			return
		}

		dataToSend := bytes.Join(entries, []byte(","))
		dataToSend = append([]byte("["), dataToSend...)
		dataToSend = append(dataToSend, []byte("]")[0])
		fmt.Println(string(dataToSend))
		res.Header().Set("Content-Type", "application/json")
		if _, err = res.Write(dataToSend); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}

		res.WriteHeader(http.StatusOK)
	}
}

type repo interface {
	SetCred(ctx context.Context, key string, data []byte) error
	SetOrder(ctx context.Context, key string, data []byte) error
	SetUserStat(ctx context.Context, key string, data []byte) error
	SetWithdraw(ctx context.Context, key string, data []byte) error
	GetCred(ctx context.Context, key string) ([]byte, error)
	GetUserStat(ctx context.Context, key string) ([]byte, error)
	GetOrderByID(ctx context.Context, key string) (string, []byte, error)
	GetOrdersByUser(ctx context.Context, key string) ([]string, [][]byte, error)
	GetWithdrawalsByUser(ctx context.Context, key string) ([]string, [][]byte, error)
	PushOrder(order repository.OrderInfo)
}
