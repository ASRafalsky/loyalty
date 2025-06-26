package handlers

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/ASRafalsky/internal/repository"
	"github.com/ASRafalsky/internal/services"
)

func RegisterPostHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		cred, status := services.ExtractCreds(req.Body)
		if status != http.StatusOK {
			res.WriteHeader(status)
			return
		}
		defer func() {
			// Handled at the logging level.
			_ = req.Body.Close()
		}()

		if status := services.SetUserCred(req.Context(), repo, &cred); status != http.StatusOK {
			res.WriteHeader(status)
			return
		}

		if err := setToken(res, req, cred.Login, time.Now().Add(15*time.Minute)); err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusOK)
	}
}

func LoginPostHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		cred, status := services.ExtractCreds(req.Body)
		if status != http.StatusOK {
			res.WriteHeader(status)
			return
		}
		defer func() {
			// Handled at the logging level.
			_ = req.Body.Close()
		}()

		if status := services.UserAuth(req.Context(), repo, &cred); status != http.StatusOK {
			res.WriteHeader(status)
			return
		}

		if err := setToken(res, req, cred.Login, time.Now().Add(15*time.Minute)); err != nil {
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
			res.WriteHeader(http.StatusBadRequest)
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
		res.WriteHeader(services.ProcessNewOrder(req.Context(), repo, buf))
	}
}

func OrdersGetHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if len(req.Header.Get("Content-Length")) > 0 && req.Header.Get("Content-Length") != "0" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		dataToSend, status := services.ListOrders(req.Context(), repo)
		if status != http.StatusOK {
			res.WriteHeader(status)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		if _, err := res.Write(dataToSend); err != nil {
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

		entry, status := services.GetUserBalance(req.Context(), repo)
		if status != http.StatusOK {
			res.WriteHeader(status)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		if _, err := res.Write(entry); err != nil {
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
		defer func() {
			// Handled at the logging level.
			_ = req.Body.Close()
		}()

		res.WriteHeader(services.NewWithdraw(req.Context(), req.Body, repo))
	}
}

func WithdrawalsGetHandler(repo repo) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if len(req.Header.Get("Content-Length")) > 0 && req.Header.Get("Content-Length") != "0" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		dataToSend, status := services.GetWithdrawals(req.Context(), repo)
		if status != http.StatusOK {
			res.WriteHeader(status)
			return
		}

		res.Header().Set("Content-Type", "application/json")
		if _, err := res.Write(dataToSend); err != nil {
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
