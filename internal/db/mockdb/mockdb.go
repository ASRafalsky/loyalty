package mockdb

import (
	"context"
	"strings"
	"sync"
)

type DB struct {
	*cache
}

type cache struct {
	mx        sync.RWMutex
	creds     map[string][]byte
	users     map[string][]byte
	orders    map[string][]byte
	withdraws map[string][]byte
}

func New() DB {
	c := cache{}
	c.creds = make(map[string][]byte)
	c.users = make(map[string][]byte)
	c.orders = make(map[string][]byte)
	c.withdraws = make(map[string][]byte)
	return DB{
		cache: &c,
	}
}

func (d DB) SetCred(_ context.Context, key string, data []byte) error {
	d.mx.Lock()
	defer d.mx.Unlock()

	d.creds[key] = data
	return nil
}

func (d DB) UpdateOrderByID(ctx context.Context, orderID string, data []byte) error {
	d.mx.Lock()
	defer d.mx.Unlock()

	for k := range d.orders {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if strings.HasSuffix(k, orderID) {
			d.orders[k] = data
			return nil
		}
	}
	return nil
}

func (d DB) SetOrder(_ context.Context, key string, data []byte) error {
	d.mx.Lock()
	defer d.mx.Unlock()

	d.orders[key] = data
	return nil
}

func (d DB) SetUserStat(_ context.Context, key string, data []byte) error {
	d.mx.Lock()
	defer d.mx.Unlock()

	d.users[key] = data
	return nil
}

func (d DB) SetWithdraw(_ context.Context, key string, data []byte) error {
	d.mx.Lock()
	defer d.mx.Unlock()

	d.withdraws[key] = data
	return nil
}

func (d DB) GetCred(_ context.Context, key string) ([]byte, error) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	return d.creds[key], nil
}

func (d DB) GetOrder(_ context.Context, key string) ([]byte, error) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	return d.orders[key], nil
}

func (d DB) GetUserStat(_ context.Context, key string) ([]byte, error) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	return d.users[key], nil
}

func (d DB) GetOrderByID(ctx context.Context, orderID string) (key string, record []byte, err error) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	for k, v := range d.orders {
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
		if strings.HasSuffix(k, orderID) {
			return k, v, nil
		}
	}
	return "", nil, nil
}

func (d DB) GetOrdersByUser(ctx context.Context, userID string) (keys []string, records [][]byte, err error) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	for k, v := range d.orders {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		if strings.HasPrefix(k, userID) {
			keys = append(keys, k)
			records = append(records, v)
		}
	}
	return keys, records, nil
}

func (d DB) GetWithdrawalsByUser(ctx context.Context, userID string) (keys []string, records [][]byte, err error) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	for k, v := range d.withdraws {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		if strings.HasPrefix(k, userID) {
			keys = append(keys, k)
			records = append(records, v)
		}
	}
	return keys, records, nil
}

func (d DB) GetWithdrawsByOrderID(ctx context.Context, orderID string) (keys string, record []byte, err error) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	for k, v := range d.withdraws {
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
		if strings.HasSuffix(k, orderID) {
			return k, v, nil
		}
	}
	return "", nil, nil
}

func (d DB) ForEachOrder(ctx context.Context, fn func(k string, v []byte) error) error {
	d.mx.RLock()
	defer d.mx.RUnlock()

	for k, v := range d.orders {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := fn(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}
