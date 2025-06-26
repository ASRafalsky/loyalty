BEGIN;

CREATE TABLE IF NOT EXISTS credentials_test (id CHAR(16) PRIMARY KEY, payload BYTEA NOT NULL);

CREATE TABLE IF NOT EXISTS users_test (id CHAR(16) PRIMARY KEY, payload JSONB);

CREATE TABLE IF NOT EXISTS orders_test (
                                           id CHAR(32) PRIMARY KEY,
    prefix CHAR(16) GENERATED ALWAYS AS (LEFT(id, 16)) STORED,
    suffix CHAR(16) GENERATED ALWAYS AS (RIGHT(id, 16)) STORED,
    payload JSONB);
CREATE INDEX IF NOT EXISTS idx_prefix ON orders_test(prefix);
CREATE INDEX IF NOT EXISTS idx_suffix ON orders_test(suffix);

CREATE TABLE IF NOT EXISTS withdraws_test (
                                              id CHAR(32) PRIMARY KEY,
    prefix CHAR(16) GENERATED ALWAYS AS (LEFT(id, 16)) STORED,
    suffix CHAR(16) GENERATED ALWAYS AS (RIGHT(id, 16)) STORED,
    payload JSONB);
CREATE INDEX IF NOT EXISTS idx_prefix ON withdraws_test(prefix);
CREATE INDEX IF NOT EXISTS idx_suffix ON withdraws_test(suffix);

COMMIT;