package storage

import (
	"context"
	"errors"
	"sync"

	"github.com/jmoiron/sqlx"
)

const schemaSqlite3 = `
CREATE TABLE IF NOT EXISTS link (
    id INTEGER PRIMARY KEY,
    key VARCHAR(64) NOT NULL,
    value text NOT NULL
)`
const schemaPostgres = `
CREATE TABLE IF NOT EXISTS link (
    id SERIAL,
    key VARCHAR(64) NOT NULL,
    value text NOT NULL
)`

type RowDatabase struct {
	ID    string `db:"id"`
	Key   string `db:"key"`
	Value string `db:"value"`
}

type DatabaseStorage struct {
	sync.RWMutex
	db *sqlx.DB
}

func NewDatabaseStorage(db *sqlx.DB) (*DatabaseStorage, error) {
	return &DatabaseStorage{db: db}, nil
}

func (c *DatabaseStorage) Migrate() error {
	var schema string

	switch c.db.DriverName() {
	case "sqlite3":
		schema = schemaSqlite3
	case "pgx":
		schema = schemaPostgres
	default:
		return errors.New("unsupported driver type")
	}
	_, err := c.db.Exec(schema)
	return err
}

func (c *DatabaseStorage) Close() error {
	err := c.db.Close()
	if err != nil {
		return err
	}
	return nil
}

func (c *DatabaseStorage) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *DatabaseStorage) Add(ctx context.Context, url string) (string, error) {
	c.Lock()
	defer c.Unlock()
	query := "INSERT INTO link(key, value) VALUES($1, $2)"

	key, err := GetURLHash(url)
	if err != nil {
		return "", err
	}

	var id string
	if err := c.db.GetContext(ctx, &id, query, key, url); err != nil {
		return "", err
	}
	return id, nil
}

func (c *DatabaseStorage) Get(ctx context.Context, key string) (string, error) {
	c.RLock()
	defer c.RUnlock()
	var row RowDatabase
	if err := c.db.GetContext(ctx, &row, "SELECT * FROM link where key=$1", key); err != nil {
		return "", err
	}
	return row.Value, nil
}
