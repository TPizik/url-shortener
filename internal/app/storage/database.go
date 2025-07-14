package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/TPizik/url-shortener/internal/app/config"
	appErrors "github.com/TPizik/url-shortener/internal/app/errors"
	"github.com/TPizik/url-shortener/internal/app/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
)

const schemaPostgres = `
CREATE TABLE IF NOT EXISTS link (
    id SERIAL,
	user_id text NOT NULL,
    key text NOT NULL,
	is_deleted boolean NOT NULL,
    value text NOT NULL UNIQUE,
		constraint cnst_link_value unique (value)
)`

type RowDatabase struct {
	ID        string `db:"id"`
	UserID    string `db:"user_id"`
	Key       string `db:"key"`
	Value     string `db:"value"`
	IsDeleted bool   `db:"is_deleted"`
}

type DatabaseStorage struct {
	sync.RWMutex
	db     *sqlx.DB
	dbpool *pgxpool.Pool
	config *config.Config
}

func NewDatabaseStorage(db *sqlx.DB, dbpoll *pgxpool.Pool, config *config.Config) (*DatabaseStorage, error) {
	return &DatabaseStorage{db: db, dbpool: dbpoll, config: config}, nil
}

func (c *DatabaseStorage) Migrate() error {
	_, err := c.db.Exec(schemaPostgres)
	return err
}

func (c *DatabaseStorage) Close() error {
	c.dbpool.Close()
	err := c.db.Close()
	if err != nil {
		return err
	}
	return nil
}

func (c *DatabaseStorage) Drop() error {
	query := "DROP TABLE link"
	_, err := c.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (c *DatabaseStorage) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *DatabaseStorage) Add(ctx context.Context, url string, userID string) (string, error) {
	c.Lock()
	defer c.Unlock()
	query := "INSERT INTO link(user_id, key, value, is_deleted) VALUES($1, $2, $3, $4) returning id"

	key, err := GetURLHash(url)
	if err != nil {
		return "", err
	}
	var id string
	var pgErr *pgconn.PgError
	err = c.db.GetContext(ctx, &id, query, userID, key, url, false)

	if errors.As(err, &pgErr) && pgErr.Code == appErrors.PgUniqueIndexErrorCode {
		key, err = c.GetURLKey(ctx, url)
		if err != nil {
			return "", err
		}
		return key, appErrors.ErrConflict
	}
	return key, nil
}

func (c *DatabaseStorage) Get(ctx context.Context, key string) (string, error) {
	c.RLock()
	defer c.RUnlock()
	var row RowDatabase
	if err := c.db.GetContext(ctx, &row, "SELECT value FROM link where key=$1", key); err != nil {
		return "", err
	}
	return row.Value, nil
}

func (c *DatabaseStorage) AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal, userID string) ([]models.URLRowShort, error) {
	shortURLs := make([]models.URLRowShort, 0)
	for _, url := range requestURLs {
		key, err := c.Add(ctx, url.OriginalURL, userID)
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

func (c *DatabaseStorage) GetURLKey(ctx context.Context, originURL string) (string, error) {
	var row RowDatabase
	if err := c.db.GetContext(ctx, &row, "SELECT key FROM link where value=$1", originURL); err != nil {
		return "", err
	}
	return row.Key, nil
}

func (c *DatabaseStorage) GetAllUserURLs(ctx context.Context, userID string) (map[string]string, error) {
	rows := make([]RowDatabase, 0)
	err := c.db.SelectContext(ctx, &rows, "SELECT id, user_id, key, value FROM link WHERE user_id=$1 order by id", userID)
	if err != nil {
		return nil, err
	}

	data := make(map[string]string)
	for i := range rows {
		row := rows[i]
		data[fmt.Sprint(row.Key)] = row.Value
	}

	return data, nil
}

func (c *DatabaseStorage) DoDeleteURLTasks(ctx context.Context, tasks []models.DeleteURLsTask) error {
	batch := &pgx.Batch{}

	query := `
		UPDATE link
		SET is_deleted = TRUE
		WHERE user_id = $1 AND key = ANY($2)
	`

	for _, task := range tasks {
		batch.Queue(
			query,
			task.UserID,
			task.ShortURLs,
		)
	}
	batchResult := c.dbpool.SendBatch(ctx, batch)
	err := batchResult.Close()
	if err != nil {
		return err
	}
	return nil
}
