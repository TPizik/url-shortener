package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/TPizik/url-shortener/internal/app/config"
	"github.com/TPizik/url-shortener/internal/app/models"
	"github.com/jmoiron/sqlx"
)

const schemaSqlite3 = `
CREATE TABLE IF NOT EXISTS link (
    id INTEGER PRIMARY KEY,
    key text NOT NULL,
    value text NOT NULL
)`
const schemaPostgres = `
CREATE TABLE IF NOT EXISTS link (
    id SERIAL,
    key text NOT NULL,
    value text NOT NULL
)`

type RowDatabase struct {
	ID    string `db:"id"`
	Key   string `db:"key"`
	Value string `db:"value"`
}

type DatabaseStorage struct {
	sync.RWMutex
	db     *sqlx.DB
	config *config.Config
}

func NewDatabaseStorage(db *sqlx.DB, config *config.Config) (*DatabaseStorage, error) {
	return &DatabaseStorage{db: db, config: config}, nil
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
	query := "INSERT INTO link(key, value) VALUES($1, $2) returning id"

	key, err := GetURLHash(url)
	if err != nil {
		return "", err
	}
	var id string
	if err := c.db.GetContext(ctx, &id, query, key, url); err != nil {
		return "", err
	}
	return key, nil
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

func (c *DatabaseStorage) AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal) ([]models.URLRowShort, error) {
	shortURLs := make([]models.URLRowShort, 0)
	for _, url := range requestURLs {
		key, err := c.Add(ctx, url.OriginalURL)
		if err != nil {
			return nil, err
		}
		shortURL := models.URLRowShort{
			CorrelationID: url.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", c.config.ShortAddr, key),
		}
		shortURLs = append(shortURLs, shortURL)
	}
	return shortURLs, nil
}
