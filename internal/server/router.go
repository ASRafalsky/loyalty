package server

import (
	"context"
	"net/http"

	"github.com/ASRafalsky/telemetry/pkg/log"
	"github.com/go-chi/chi/v5"

	"github.com/ASRafalsky/internal/server/handlers"
	"github.com/ASRafalsky/internal/server/middleware"
)

func newRouter(repo dataRepository, logger *log.Logger) http.Handler {
	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Route("/api", func(r chi.Router) {
			r.Route("/user", func(r chi.Router) {
				r.Post("/register", middleware.WithCompress(handlers.RegisterPostHandler(repo), logger))
				r.Post("/login", middleware.WithCompress(handlers.LoginPostHandler(repo), logger))
				r.Post("/orders",
					middleware.WithAuth(middleware.WithCompress(handlers.OrdersPostHandler(repo), logger)))
				r.Get("/orders",
					middleware.WithAuth(middleware.WithCompress(handlers.OrdersGetHandler(repo), logger)))
				r.Get("/withdrawals",
					middleware.WithAuth(middleware.WithCompress(handlers.WithdrawalsGetHandler(repo), logger)))
			})
			r.Route("/balance", func(r chi.Router) {
				r.Get("/", middleware.WithAuth(middleware.WithCompress(handlers.BalanceGetHandler(repo), logger)))
				r.Post("/withdraw",
					middleware.WithAuth(middleware.WithCompress(handlers.WithdrawPostHandler(repo), logger)))
			})
		})
	})
	return r
}

type dataRepository interface {
	SetCred(ctx context.Context, key string, data []byte) error
	SetOrder(ctx context.Context, key string, data []byte) error
	SetUserStat(ctx context.Context, key string, data []byte) error
	GetCred(ctx context.Context, key string) ([]byte, error)
	GetUserStat(ctx context.Context, key string) ([]byte, error)
	GetOrderByID(ctx context.Context, k string) (string, []byte, error)
	GetOrdersByUser(ctx context.Context, k string) ([]string, [][]byte, error)
	GetWithdrawalsByUser(ctx context.Context, k string) ([]string, [][]byte, error)
}
