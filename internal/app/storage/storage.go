package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/TPizik/url-shortener/internal/app/config"
	appErrors "github.com/TPizik/url-shortener/internal/app/errors"
	"github.com/TPizik/url-shortener/internal/app/models"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type StorageExpected interface {
	Get(ctx context.Context, key string) (string, error)
	Add(ctx context.Context, key string, userID string) (string, error)
	AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal, userID string) ([]models.URLRowShort, error)
	GetAllUserURLs(ctx context.Context, userID string) (map[string]string, error)
	DoDeleteURLTasks(ctx context.Context, tasks []models.DeleteURLsTask) error
	Ping(ctx context.Context) error
	Close() error
	Drop() error
}

type Storage struct {
	storage StorageExpected
}

func NewStorage(ctx context.Context, config *config.Config) (*Storage, error) {
	switch {
	case config.DBDSN != "":
		db, err := sqlx.Open("pgx", config.DBDSN)
		if err != nil {
			return nil, err
		}
		dbpoll, err := pgxpool.New(ctx, config.DBDSN)
		if err != nil {
			return nil, err
		}
		storage, err := NewDatabaseStorage(db, dbpoll, config)
		if err != nil {
			return nil, err
		}
		err = storage.Migrate()
		if err != nil {
			return nil, err
		}
		return &Storage{storage: storage}, nil
	case config.FileStoragePath != "":
		storage, err := NewFileStorage(config.FileStoragePath, config)
		if err != nil {
			return nil, err
		}
		err = storage.Load()
		if err != nil {
			return nil, err
		}
		return &Storage{storage: storage}, nil
	default:
		storage := NewInmemoryStorage(config)
		return &Storage{storage: storage}, nil
	}
}

func (c *Storage) Ping(ctx context.Context) error {
	return c.storage.Ping(ctx)
}

func (c *Storage) Close() error {
	return c.storage.Close()
}

func (c *Storage) Drop() error {
	return c.storage.Drop()
}

func (c *Storage) Add(ctx context.Context, url string, userID string) (string, error) {
	key, err := c.storage.Add(ctx, url, userID)
	if err != nil && err == appErrors.ErrConflict {
		return key, err
	}
	if err != nil {
		return "", err
	}

	return key, nil
}

func (c *Storage) Get(ctx context.Context, key string) (string, error) {
	url, err := c.storage.Get(ctx, key)
	if err != nil {
		return "", err
	}

	return url, nil
}

func (c *Storage) AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal, userID string) ([]models.URLRowShort, error) {
	url, err := c.storage.AddByBatch(ctx, requestURLs, userID)
	if err != nil {
		return nil, err
	}

	return url, nil
}

func (c *Storage) GetAllUserURLs(ctx context.Context, userID string) (map[string]string, error) {
	urls, err := c.storage.GetAllUserURLs(ctx, userID)
	if err != nil {
		return nil, err
	}

	return urls, nil
}

func GetURLHash(url string) (string, error) {
	h := sha256.New()
	_, err := h.Write([]byte(url))
	if err != nil {
		return "", appErrors.ErrWrite
	}

	sha256Sum := h.Sum(nil)
	key := hex.EncodeToString(sha256Sum[:5])

	return key, nil
}

func (c *Storage) DoDeleteURLTasks(ctx context.Context, tasks []models.DeleteURLsTask) error {
	err := c.storage.DoDeleteURLTasks(ctx, tasks)
	if err != nil {
		return err
	}
	return nil
}
