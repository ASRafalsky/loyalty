package postgres

import (
	"context"
	"errors"
	"math/rand/v2"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ASRafalsky/telemetry/pkg/log"
	"github.com/golang-migrate/migrate/v4"
	"github.com/mailru/easyjson"
	"github.com/stretchr/testify/require"

	"github.com/ASRafalsky/internal/config"
	"github.com/ASRafalsky/internal/models"
)

func TestCredsDB(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationFilesPath := filepath.Dir(file) + "/test_migrations"
	ctx := context.Background()

	const (
		cnt           = 200
		insertCred    = `INSERT INTO credentials_test (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`
		selectCred    = `SELECT payload FROM credentials_test WHERE id = $1`
		deleteCred    = `DELETE FROM credentials_test WHERE id = $1`
		selectForEach = `SELECT id, payload FROM credentials_test WHERE id > $1 ORDER BY id LIMIT $2`
		insertBatch   = `INSERT INTO credentials_test (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`
	)

	Log, err := log.AddLoggerWith("INFO", "")
	require.NoError(t, err)
	defer Log.Sync()
	cfg := config.DB{
		DSN:             "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
		MigrationsTable: "",
		MigrationsPath:  migrationFilesPath,
	}
	db, err := InitDB(context.Background(), cfg, Log)
	require.NoError(t, err)
	credsList := make([]models.Credential, cnt)
	keys := make([]string, cnt)
	for i := range cnt {
		credsList[i] = models.Credential{
			Login:    "login" + strconv.Itoa(i),
			Password: "password" + strconv.Itoa(rand.Int()),
		}
		keys[i], err = models.IDHash(credsList[i].Login)
		require.NoError(t, err)
	}

	t.Run("Set", func(t *testing.T) {
		for i := range credsList {
			require.NoError(t, db.set(ctx, insertCred, keys[i], []byte(credsList[i].Password)))
		}
	})

	t.Run("Get", func(t *testing.T) {
		for i := range credsList {
			res, err := db.get(ctx, selectCred, keys[i])
			require.NoError(t, err)
			require.Equal(t, credsList[i].Password, string(res))
		}
	})

	t.Run("Get_by_invalid_key", func(t *testing.T) {
		_, err := db.get(ctx, selectCred, "kek")
		require.NoError(t, err)
	})

	t.Run("Update", func(t *testing.T) {
		for i := range credsList {
			res, err := db.get(ctx, selectCred, keys[i])
			require.NoError(t, err)
			require.NoError(t, db.set(ctx, insertCred, keys[i], res))
		}
	})

	t.Run("ForEach", func(t *testing.T) {
		require.NoError(t, db.forEach(ctx, selectForEach, func(k string, v []byte) error {
			require.Contains(t, keys, k)
			return nil
		}))
	})

	t.Run("SetBatch", func(t *testing.T) {
		_, err = db.ExecContext(ctx, `DELETE FROM credentials_test`)
		require.NoError(t, err)
		pairs := make([]Pair, cnt)
		for i := range credsList {
			pairs[i] = Pair{
				ID:      keys[i],
				Payload: []byte(credsList[i].Password),
			}
		}
		require.NoError(t, db.setBatch(ctx, insertBatch, pairs))

		for i := range credsList {
			res, err := db.get(ctx, selectCred, keys[i])
			require.NoError(t, err)
			require.Equal(t, credsList[i].Password, string(res))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		for i := range keys {
			res, err := db.get(ctx, selectCred, keys[i])
			require.NoError(t, err)
			require.NotNil(t, res)
			err = db.deleteEntrie(ctx, deleteCred, keys[i])
			require.NoError(t, err)
			res, err = db.get(ctx, selectCred, keys[i])
			require.NoError(t, err)
			require.Nil(t, res)
		}
	})
}

func TestOrdersDB(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	migrationFilesPath := filepath.Dir(file) + "/test_migrations"

	db, err := Open("postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Skip("Failed to connect to database: ", err.Error())
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Log("Failed to close db:", err.Error())
		}
	}()

	ctx := context.Background()
	if err = db.WaitDBIsReady(ctx, 5, time.Second); err != nil {
		t.Skip("Failed to connect to database: ", err.Error())
	}

	const (
		cnt           = 200
		insertOrder   = `INSERT INTO orders_test (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`
		selectOrder   = `SELECT payload FROM orders_test WHERE id = $1`
		deleteOrder   = `DELETE FROM orders_test WHERE id = $1`
		selectForEach = `SELECT id, payload FROM orders_test WHERE id > $1 ORDER BY id LIMIT $2`
		insertBatch   = `INSERT INTO orders_test (id, payload) VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET payload = $2`

		selectOrdersByUser = `SELECT id, payload
								FROM orders_test
								WHERE prefix = $1
								ORDER BY (payload->>'uploaded_at')::timestamp DESC;`
		selectOrdersByOrder = `SELECT id, payload
								FROM orders_test
								WHERE suffix = $1;`
		updateOrderByOrder = `UPDATE orders_test SET payload = $2 WHERE suffix = $1`
	)

	if err = db.MigrateUp(migrationFilesPath); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			require.NoError(t, err)
		}
	}
	defer func() {
		if err = db.MigrateRollback(migrationFilesPath, 1); err != nil {
			require.NoError(t, err)
		}
	}()

	ordersList := make([]models.Order, cnt)
	keys := make([]string, cnt)
	for i := range cnt {
		ordersList[i] = models.Order{
			ID: strconv.Itoa(i),
			TS: time.Now(),
		}
		loginID, err := models.IDHash("login" + strconv.Itoa(i))
		require.NoError(t, err)
		orderID, err := models.IDHash(ordersList[i].ID)
		require.NoError(t, err)
		keys[i] = loginID + orderID
	}

	t.Run("Set", func(t *testing.T) {
		for i := range ordersList {
			data, err := easyjson.Marshal(ordersList[i])
			require.NoError(t, err)
			require.NoError(t, db.set(ctx, insertOrder, keys[i], data))
		}
	})

	t.Run("Get", func(t *testing.T) {
		for i := range ordersList {
			res, err := db.get(ctx, selectOrder, keys[i])
			require.NoError(t, err)
			var order models.Order
			require.NoError(t, easyjson.Unmarshal(res, &order))
			require.Equal(t, ordersList[i].Status, order.Status)
			require.Equal(t, ordersList[i].ID, order.ID)
			require.Equal(t, ordersList[i].Points, order.Points)
			require.Equal(t, ordersList[i].TS.Format(time.RFC3339), order.TS.Format(time.RFC3339))
		}
	})

	t.Run("Update", func(t *testing.T) {
		for i := range ordersList {
			res, err := db.get(ctx, selectOrder, keys[i])
			require.NoError(t, err)
			require.NoError(t, db.set(ctx, insertOrder, keys[i], res))
			require.NoError(t, db.set(ctx, updateOrderByOrder, keys[i][16:], res))
		}
	})

	t.Run("UpdateByOrderID", func(t *testing.T) {
		for i := range ordersList {
			res, err := db.get(ctx, selectOrder, keys[i])
			require.NoError(t, err)
			require.NoError(t, db.set(ctx, updateOrderByOrder, keys[i][16:], res))
		}
	})

	t.Run("ForEach", func(t *testing.T) {
		require.NoError(t, db.forEach(ctx, selectForEach, func(k string, v []byte) error {
			require.Contains(t, keys, k)
			return nil
		}))
	})

	t.Run("SetBatch", func(t *testing.T) {
		_, err = db.ExecContext(ctx, `DELETE FROM orders_test`)
		require.NoError(t, err)
		pairs := make([]Pair, cnt)
		for i := range ordersList {
			data, err := easyjson.Marshal(ordersList[i])
			require.NoError(t, err)
			pairs[i] = Pair{
				ID:      keys[i],
				Payload: data,
			}
		}
		require.NoError(t, db.setBatch(ctx, insertBatch, pairs))

		for i := range ordersList {
			res, err := db.get(ctx, selectOrder, keys[i])
			require.NoError(t, err)
			var order models.Order
			require.NoError(t, easyjson.Unmarshal(res, &order))
			require.Equal(t, ordersList[i].Status, order.Status)
			require.Equal(t, ordersList[i].ID, order.ID)
			require.Equal(t, ordersList[i].Points, order.Points)
			require.Equal(t, ordersList[i].TS.Format(time.RFC3339), order.TS.Format(time.RFC3339))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		for i := range keys {
			res, err := db.get(ctx, selectOrder, keys[i])
			require.NoError(t, err)
			require.NotNil(t, res)
			err = db.deleteEntrie(ctx, deleteOrder, keys[i])
			require.NoError(t, err)
			res, err = db.get(ctx, selectOrder, keys[i])
			require.NoError(t, err)
			require.Nil(t, res)
		}
	})

	prefixes := make([]string, cnt)
	prefixCnt := make(map[string]int)
	for i := range cnt {
		prefixes[i], err = models.IDHash("login" + strconv.Itoa(i%5))
		prefixCnt[prefixes[i]]++
		ordersList[i] = models.Order{
			ID: strconv.Itoa(i),
			TS: time.Now(),
		}
		time.Sleep(100 * time.Millisecond)
		orderID, err := models.IDHash(ordersList[i].ID)
		require.NoError(t, err)
		keys[i] = prefixes[i] + orderID
	}
	slices.Sort(prefixes)
	prefixes = slices.Compact(prefixes)

	for i := range ordersList {
		data, err := easyjson.Marshal(ordersList[i])
		require.NoError(t, err)
		require.NoError(t, db.set(ctx, insertOrder, keys[i], data))
	}

	t.Run("GetByPrefix", func(t *testing.T) {
		for _, prefix := range prefixes {
			keyList, recordList, err := db.getRecordsBy(ctx, selectOrdersByUser, prefix)
			require.NoError(t, err)
			require.Len(t, recordList, prefixCnt[prefix])
			var previousRecordTime time.Time
			for j := range recordList {
				require.True(t, strings.HasPrefix(keyList[j], prefix))
				var order models.Order
				require.NoError(t, easyjson.Unmarshal(recordList[j], &order))
				orderIDHash, err := models.IDHash(order.ID)
				require.NoError(t, err)
				require.True(t, strings.HasSuffix(keyList[j], orderIDHash))
				if !previousRecordTime.IsZero() {
					require.True(t, previousRecordTime.After(order.TS))
				}
				previousRecordTime = order.TS.Round(time.Second)
			}
		}
	})

	t.Run("GetBySuffix", func(t *testing.T) {
		for _, order := range ordersList {
			orderIDHash, err := models.IDHash(order.ID)
			require.NoError(t, err)
			keyList, recordList, err := db.getRecordsBy(ctx, selectOrdersByOrder, orderIDHash)
			require.NoError(t, err)
			require.Len(t, keyList, 1)
			require.Len(t, recordList, 1)
			require.True(t, strings.HasSuffix(keyList[0], orderIDHash))
			var orderRecord models.Order
			require.NoError(t, easyjson.Unmarshal(recordList[0], &orderRecord))
			orderRecordIDHash, err := models.IDHash(order.ID)
			require.NoError(t, err)
			require.Equal(t, orderIDHash, orderRecordIDHash)
		}
	})
}
