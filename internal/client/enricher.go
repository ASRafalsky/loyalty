package client

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/ASRafalsky/telemetry/pkg/log"
	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/mailru/easyjson"
	"go.uber.org/multierr"
	"golang.org/x/time/rate"

	"github.com/ASRafalsky/internal/config"
	"github.com/ASRafalsky/internal/models"
	"github.com/ASRafalsky/internal/repository"
)

type Enricher struct {
	addr    string
	storage dataRepository
	client  *httpclient.Client
}

func NewEnricher(cfg config.Config, repo dataRepository) *Enricher {
	return &Enricher{
		storage: repo,
		addr:    cfg.AccrualAddr,
		client:  NewClient(),
	}
}

func (e *Enricher) loadUnenrichedOrders(ctx context.Context) error {
	err := e.storage.ForEachOrder(ctx, func(k string, v []byte) error {
		var order models.Order
		if err := easyjson.Unmarshal(v, &order); err != nil {
			return err
		}
		if order.Status != "PROCESSED" && order.Status != "INVALID" {
			e.storage.PushOrder(repository.OrderInfo{
				User:  k[:16],
				Order: order,
			})
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (e *Enricher) Run(ctx context.Context, logger *log.Logger) error {
	if err := e.loadUnenrichedOrders(ctx); err != nil {
		return err
	}

	go e.serve(ctx, logger)

	return nil
}

func (e *Enricher) serve(ctx context.Context, logger *log.Logger) {
	sendTimer := time.NewTicker(time.Second * 1)
	defer sendTimer.Stop()

	limit := rate.Every(time.Second / time.Duration(500))

	rl := rate.NewLimiter(limit, 1)
	for {
		select {
		case <-ctx.Done():
			return
		case <-sendTimer.C:
			if !rl.Allow() {
				continue
			}
			sendCtx, cancel := context.WithTimeout(ctx, time.Millisecond*100) // I'm so sorry)).

			if err := withRetryOnErr(sendCtx, 3, func() error {
				return e.sendOrder(ctx)
			}); err != nil {
				logger.Error("[accrual] failed to process order", err.Error())
			}
			cancel()
		}
	}
}

func (e *Enricher) sendOrder(ctx context.Context) (err error) {
	header := http.Header{
		"Content-Type": []string{"text/plain"},
	}

	order, ok := e.storage.PopOrder()
	if !ok {
		return nil
	}
	urlData := e.addr + "/api/orders/" + order.Order.ID
	resp, err := e.client.Get(urlData, header)
	if err != nil {
		return err
	}
	defer func() {
		if errClose := resp.Body.Close(); errClose != nil {
			err = multierr.Append(err, errClose)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("unexpected content type: %s", resp.Header.Get("Content-Type"))
	}
	var buf []byte
	if resp.Header.Get("Content-Encoding") == "gzip" {
		zr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		defer func() {
			if errZr := zr.Close(); errZr != nil {
				err = multierr.Append(err, errZr)
			}
		}()
		buf, err = io.ReadAll(zr)
		if err != nil && err != io.EOF {
			return err
		}
	} else {
		buf, err = io.ReadAll(resp.Body)
		if err != nil && err != io.EOF {
			return err
		}
	}

	if len(buf) == 0 {
		return nil
	}

	var accrual models.Accrual
	if err = easyjson.Unmarshal(buf, &accrual); err != nil {
		return err
	}
	err = e.updateOrder(ctx, order.Order, accrual)
	if err != nil {
		return err
	}

	if accrual.Status != "PROCESSED" && accrual.Status != "INVALID" {
		e.storage.PushOrder(order)
		return nil
	}

	err = e.updateUserStat(ctx, order.User, accrual)
	if err != nil {
		return err
	}
	return nil
}

func (e *Enricher) updateOrder(ctx context.Context, order models.Order, accrual models.Accrual) error {
	if order.Status != accrual.Status {
		order.Status = accrual.Status

		if accrual.Points != nil {
			points := *accrual.Points
			order.Points = &points
		}

		idHash, err := models.IDHash(order.ID)
		if err != nil {
			return err
		}
		data, err := easyjson.Marshal(order)
		if err != nil {
			return err
		}
		if err = e.storage.UpdateOrderByID(ctx, idHash, data); err != nil {
			return err
		}
	}
	return nil
}

func (e *Enricher) updateUserStat(ctx context.Context, userID string, accrual models.Accrual) error {
	if accrual.Points != nil {
		stat, err := e.storage.GetUserStat(ctx, userID)
		if err != nil {
			return err
		}

		var userStat models.UserStats
		if len(stat) > 0 {
			if err = easyjson.Unmarshal(stat, &userStat); err != nil {
				return err
			}
		}

		if userStat.CurrentPoints == nil {
			points := *accrual.Points
			userStat.CurrentPoints = &points
		} else {
			*userStat.CurrentPoints += *accrual.Points
		}

		buf, err := easyjson.Marshal(userStat)
		if err != nil {
			return err
		}
		err = e.storage.SetUserStat(ctx, userID, buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func withRetryOnErr(ctx context.Context, cnt int, fn func() error) error {
	cnt--
	err := fn()
	if err != nil && (errors.Is(err, syscall.ECONNREFUSED) ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "unreachable") ||
		strings.Contains(err.Error(), "no route to host") ||
		strings.Contains(err.Error(), "invalid argument")) {
		wait := 1
		ticker := time.NewTicker(time.Duration(wait) * time.Second)
		defer ticker.Stop()
		for cnt >= 1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				err = fn()
				if err == nil {
					return nil
				}
				wait += 2
				ticker.Reset(time.Duration(wait) * time.Second)
				cnt--
			}
		}
	}
	return err
}

type dataRepository interface {
	SetOrder(ctx context.Context, key string, data []byte) error
	SetUserStat(ctx context.Context, key string, data []byte) error
	GetUserStat(ctx context.Context, key string) ([]byte, error)
	GetOrder(ctx context.Context, key string) ([]byte, error)
	GetOrderByID(ctx context.Context, key string) (string, []byte, error)
	ForEachOrder(ctx context.Context, fn func(k string, v []byte) error) error
	UpdateOrderByID(ctx context.Context, orderID string, data []byte) error
	PopOrder() (repository.OrderInfo, bool)
	PushOrder(order repository.OrderInfo)
}
