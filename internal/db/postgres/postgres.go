package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/multierr"
)

type DB struct {
	*sql.DB
	m migrate.Migrate
}

var (
	ErrConnectionIssue = errors.New("database connection issue")
	ErrTransaction     = errors.New("database transaction issue")
)

func Open(param string) (DB, error) {
	db, err := sql.Open("pgx", param)
	return DB{DB: db}, err
}
func (d DB) Close() error {
	return d.DB.Close()
}

func (d DB) Ping(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}

// Bootstrap prepares DB.
func (d DB) Bootstrap(ctx context.Context) error {
	if err := d.bootstrap(ctx, `CREATE TABLE IF NOT EXISTS credentials (
            id CHAR(16) PRIMARY KEY,
            payload BYTEA NOT NULL,);`); err != nil {
		return err
	}
	if err := d.bootstrap(ctx, `CREATE TABLE IF NOT EXISTS users (
            id CHAR(16) PRIMARY KEY,
            payload JSONB,);`); err != nil {
		return err
	}
	if err := d.bootstrap(ctx, `CREATE TABLE IF NOT EXISTS orders (
            id CHAR(32) PRIMARY KEY,
    		prefix CHAR(16) GENERATED ALWAYS AS (LEFT(id, 16)) STORED,
			suffix CHAR(16) GENERATED ALWAYS AS (RIGHT(id, 16)) STORED,
            payload JSONB,);
 			CREATE INDEX IF NOT EXISTS idx_prefix ON orders
 			orders(prefix);
			CREATE INDEX IF NOT EXISTS idx_suffix ON orders
 			orders(suffix);`); err != nil {
		return err
	}
	return nil
}

func (d DB) bootstrap(ctx context.Context, query string) (err error) {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return
	}

	defer func() {
		if errRollback := tx.Rollback(); err != nil && errRollback != nil {
			err = multierr.Append(err, errRollback)
		}
	}()

	// Create metrics table.
	if _, errCreate := tx.ExecContext(ctx, query); err != nil {
		err = multierr.Append(err, errCreate)
	}

	if errCommit := tx.Commit(); errCommit != nil {
		err = multierr.Append(err, errCommit)
	}
	return
}

func (d DB) SetCred(ctx context.Context, key string, data []byte) error {
	if len(key) != 16 {
		return errors.New("invalid key length")
	}
	return d.set(ctx,
		`INSERT INTO credentials (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`, key, data)
}

func (d DB) SetUserStat(ctx context.Context, key string, data []byte) error {
	if len(key) != 16 {
		return errors.New("invalid key length")
	}
	return d.set(ctx,
		`INSERT INTO users (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`, key, data)
}

func (d DB) SetOrder(ctx context.Context, key string, data []byte) error {
	if len(key) != 32 {
		return errors.New("invalid key length")
	}
	return d.set(ctx,
		`INSERT INTO orders (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`, key, data)
}

func (d DB) SetWithdraw(ctx context.Context, key string, data []byte) error {
	if len(key) != 32 {
		return errors.New("invalid key length")
	}
	return d.set(ctx,
		`INSERT INTO withdraw (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`, key, data)
}

func (d DB) set(ctx context.Context, query, key string, data []byte) error {
	_, err := d.ExecContext(ctx, query, key, data)
	return errorHandler(err)
}

func (d DB) GetCred(ctx context.Context, key string) ([]byte, error) {
	if len(key) != 16 {
		return nil, errors.New("invalid key length")
	}
	payload, err := d.get(ctx, `SELECT payload FROM credentials WHERE id = $1`, key)
	return payload, err
}

func (d DB) GetUserStat(ctx context.Context, key string) ([]byte, error) {
	if len(key) != 16 {
		return nil, errors.New("invalid key length")
	}
	payload, err := d.get(ctx, `SELECT payload FROM users WHERE id = $1`, key)
	return payload, err
}

func (d DB) GetOrder(ctx context.Context, key string) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("invalid key length")
	}
	payload, err := d.get(ctx, `SELECT payload FROM orders WHERE id = $1`, key)
	return payload, err
}

func (d DB) GetWithdraw(ctx context.Context, key string) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("invalid key length")
	}
	payload, err := d.get(ctx, `SELECT payload FROM withdraw WHERE id = $1`, key)
	return payload, err
}

func (d DB) get(ctx context.Context, query, key string) ([]byte, error) {
	var (
		err     error
		payload []byte
	)

	err = d.QueryRowContext(ctx, query, key).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, errorHandler(err)
	}
	return payload, nil
}

func (d DB) GetOrdersByUser(ctx context.Context, userID string) (keys []string, records [][]byte, err error) {
	query := `
        SELECT id, payload 
        FROM orders 
        WHERE prefix = $1
        ORDER BY (payload->>'uploaded_at')::timestamp DESC;`
	return d.getRecordsBy(ctx, query, userID)
}

func (d DB) GetOrderByID(ctx context.Context, orderID string) (key string, record []byte, err error) {
	query := `
        SELECT id, payload 
        FROM orders 
        WHERE suffix = $1;`
	keys, records, err := d.getRecordsBy(ctx, query, orderID)
	if err != nil {
		return "", nil, err
	}
	if len(records) == 0 {
		return "", nil, nil
	}
	return keys[0], records[0], nil
}

func (d DB) GetWithdrawalsByUser(ctx context.Context, userID string) (keys []string, records [][]byte, err error) {
	query := `
        SELECT id, payload 
        FROM withdraws 
        WHERE prefix = $1
        ORDER BY (payload->>'uploaded_at')::timestamp DESC;`
	return d.getRecordsBy(ctx, query, userID)
}

func (d DB) GetWithdrawsByOrderID(ctx context.Context, orderID string) (keys []string, records [][]byte, err error) {
	query := `
        SELECT id, payload 
        FROM withdraws 
        WHERE suffix = $1;`
	return d.getRecordsBy(ctx, query, orderID)
}

func (d DB) getRecordsBy(ctx context.Context, query, id string) (keys []string, records [][]byte, err error) {

	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if errRollback := tx.Rollback(); err != nil && errRollback != nil {
			err = multierr.Append(err, errRollback)
		}
	}()
	rows, err := d.Query(query, id)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = multierr.Append(err, closeErr)
		}
	}()

	for rows.Next() {
		var (
			record []byte
			key    string
		)
		if err = rows.Scan(&key, &record); err != nil {
			return nil, nil, err
		}
		records = append(records, record)
		keys = append(keys, key)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, err
	}
	if errCommit := tx.Commit(); errCommit != nil {
		err = multierr.Append(err, errCommit)
		return nil, nil, err
	}

	return keys, records, nil
}

func (d DB) DeleteCred(ctx context.Context, key string) error {
	return d.deleteEntrie(ctx, `DELETE FROM credentials WHERE id = $1`, key)
}

func (d DB) DeleteUserStat(ctx context.Context, key string) error {
	return d.deleteEntrie(ctx, `DELETE FROM users WHERE id = $1`, key)
}

func (d DB) DeleteOrder(ctx context.Context, key string) error {
	return d.deleteEntrie(ctx, `DELETE FROM orders WHERE id = $1`, key)
}

func (d DB) deleteEntrie(ctx context.Context, query, key string) error {
	_, err := d.ExecContext(ctx, query, key)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return errorHandler(err)
}

func (d DB) ForEachCred(ctx context.Context, fn func(k string, v []byte) error) error {
	return errorHandler(d.forEach(ctx, `SELECT id, payload FROM credentials ORDER BY id LIMIT $1 OFFSET $2`, fn))
}

func (d DB) ForEachUserStat(ctx context.Context, fn func(k string, v []byte) error) error {
	return errorHandler(d.forEach(ctx, `SELECT id, payload FROM users ORDER BY id LIMIT $1 OFFSET $2`, fn))
}

func (d DB) ForEachOrder(ctx context.Context, fn func(k string, v []byte) error) error {
	return errorHandler(d.forEach(ctx, `SELECT id, payload FROM orders ORDER BY id LIMIT $1 OFFSET $2`, fn))
}

const BatchSz = 1000

func (d DB) forEach(ctx context.Context, query string, fn func(k string, v []byte) error) (err error) {
	offset := 0
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if errRollback := tx.Rollback(); err != nil && errRollback != nil {
			err = multierr.Append(err, errRollback)
		}
	}()

	for {
		rows, err := tx.QueryContext(ctx, query, BatchSz, offset)
		if err != nil {
			return err
		}

		var processed int
		for rows.Next() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			var id string
			var payload []byte
			if err = rows.Scan(&id, &payload); err != nil {
				if closeErr := rows.Close(); closeErr != nil {
					err = multierr.Append(err, closeErr)
				}
				return err
			}
			if err = fn(id, payload); err != nil {
				if closeErr := rows.Close(); closeErr != nil {
					err = multierr.Append(err, closeErr)
				}
				return err
			}
			processed++
		}
		if err = rows.Err(); err != nil {
			if closeErr := rows.Close(); closeErr != nil {
				err = multierr.Append(err, closeErr)
			}
			return err
		}
		err = rows.Close()
		if processed < BatchSz {
			if errCommit := tx.Commit(); err != nil {
				err = multierr.Append(err, errCommit)
			}
			return err
		}
		offset += BatchSz
	}
}

func (d DB) SizeCred() (int, error) {
	return d.size(`SELECT COUNT(*) FROM credentials`)
}

func (d DB) SizeUserStat() (int, error) {
	return d.size(`SELECT COUNT(*) FROM users`)
}

func (d DB) SizeOrders() (int, error) {
	return d.size(`SELECT COUNT(*) FROM orders`)
}

func (d DB) size(query string) (int, error) {
	var sz int
	err := d.QueryRow(query).Scan(&sz)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, errorHandler(err)
	}
	return sz, nil
}

type Pair struct {
	ID      string
	Payload []byte
}

func (d DB) SetBatchCred(ctx context.Context, batch []Pair) error {
	return errorHandler(d.setBatch(ctx,
		`INSERT INTO credentials (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`, batch))
}

func (d DB) SetBatchUserStat(ctx context.Context, batch []Pair) error {
	return errorHandler(d.setBatch(ctx,
		`INSERT INTO users (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`, batch))
}

func (d DB) SetBatchOrders(ctx context.Context, batch []Pair) error {
	return errorHandler(d.setBatch(ctx,
		`INSERT INTO orders (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`, batch))
}

func (d DB) setBatch(ctx context.Context, query string, batch []Pair) (err error) {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if errRollback := tx.Rollback(); err != nil && errRollback != nil {
			err = multierr.Append(err, errRollback)
		}
	}()

	for i := range batch {
		_, err = tx.ExecContext(ctx, query, batch[i].ID, batch[i].Payload)
		if err != nil {
			return err
		}
	}
	err = tx.Commit()
	return
}

func errorHandler(err error) error {
	var pgErr *pgconn.PgError
	if err != nil && errors.As(err, &pgErr) {
		switch {
		case pgerrcode.IsInvalidTransactionInitiation(pgErr.Code),
			pgerrcode.IsInvalidTransactionState(pgErr.Code),
			pgerrcode.IsInvalidTransactionTermination(pgErr.Code):
			return ErrTransaction
		case pgerrcode.IsConnectionException(pgErr.Code):
			return ErrConnectionIssue

		}
	}
	return err
}
