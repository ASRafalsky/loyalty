BEGIN;

CREATE TABLE IF NOT EXISTS credentials (id CHAR(16) PRIMARY KEY, payload BYTEA NOT NULL);

CREATE TABLE IF NOT EXISTS users (id CHAR(16) PRIMARY KEY, payload JSONB);

CREATE TABLE IF NOT EXISTS orders (
                                           id CHAR(32) PRIMARY KEY,
    prefix CHAR(16) GENERATED ALWAYS AS (LEFT(id, 16)) STORED,
    suffix CHAR(16) GENERATED ALWAYS AS (RIGHT(id, 16)) STORED,
    payload JSONB);
CREATE INDEX IF NOT EXISTS idx_prefix ON orders(prefix);
CREATE INDEX IF NOT EXISTS idx_suffix ON orders(suffix);

CREATE TABLE IF NOT EXISTS withdraws (
                                              id CHAR(32) PRIMARY KEY,
    prefix CHAR(16) GENERATED ALWAYS AS (LEFT(id, 16)) STORED,
    suffix CHAR(16) GENERATED ALWAYS AS (RIGHT(id, 16)) STORED,
    payload JSONB);
CREATE INDEX IF NOT EXISTS idx_prefix ON withdraws(prefix);
CREATE INDEX IF NOT EXISTS idx_suffix ON withdraws(suffix);

COMMIT;