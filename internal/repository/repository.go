package repository

import (
	"container/list"
	"context"
	"sync"

	"github.com/ASRafalsky/internal/models"
)

type ExtendedRepository struct {
	db
	c cache
}

type cache struct {
	mx    sync.RWMutex
	queue *list.List
}

type OrderInfo struct {
	User  string
	Order models.Order
}

func NewExtendedRepository(db db) *ExtendedRepository {
	return &ExtendedRepository{
		db: db,
		c: cache{
			queue: list.New(),
		},
	}
}

func (r *ExtendedRepository) PushOrder(order OrderInfo) {
	r.c.mx.Lock()
	defer r.c.mx.Unlock()

	r.c.queue.PushBack(order)
}

func (r *ExtendedRepository) PopOrder() (OrderInfo, bool) {
	r.c.mx.RLock()
	defer r.c.mx.RUnlock()

	e := r.c.queue.Front()
	if e == nil {
		return OrderInfo{}, false
	}
	r.c.queue.Remove(e)
	return e.Value.(OrderInfo), true
}

type db interface {
	SetCred(ctx context.Context, key string, data []byte) error
	GetCred(ctx context.Context, key string) ([]byte, error)
	GetOrdersByUser(ctx context.Context, k string) ([]string, [][]byte, error)
	SetWithdraw(ctx context.Context, key string, data []byte) error
	GetWithdrawalsByUser(ctx context.Context, k string) ([]string, [][]byte, error)
	SetOrder(ctx context.Context, key string, data []byte) error
	SetUserStat(ctx context.Context, key string, data []byte) error
	GetUserStat(ctx context.Context, key string) ([]byte, error)
	GetOrder(ctx context.Context, key string) ([]byte, error)
	GetOrderByID(ctx context.Context, key string) (string, []byte, error)
	ForEachOrder(ctx context.Context, fn func(k string, v []byte) error) error
	UpdateOrderByID(ctx context.Context, orderID string, data []byte) error
}
