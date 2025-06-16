package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/TPizik/url-shortener/internal/app/config"
	appErrors "github.com/TPizik/url-shortener/internal/app/errors"
	"github.com/TPizik/url-shortener/internal/app/models"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
)

type StorageExpected interface {
	Get(ctx context.Context, key string) (string, error)
	Add(ctx context.Context, key string) (string, error)
	AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal) ([]models.URLRowShort, error)
	Ping(ctx context.Context) error
	Close() error
}

type Storage struct {
	storage StorageExpected
}

func NewStorage(config *config.Config) (*Storage, error) {
	switch {
	case config.DBDSN != "":
		db, err := sqlx.Open("pgx", config.DBDSN)
		if err != nil {
			return nil, err
		}
		storage, err := NewDatabaseStorage(db, config)
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

func (c *Storage) Add(ctx context.Context, url string) (string, error) {
	key, err := c.storage.Add(ctx, url)
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

func (c *Storage) AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal) ([]models.URLRowShort, error) {
	url, err := c.storage.AddByBatch(ctx, requestURLs)
	if err != nil {
		return nil, err
	}

	return url, nil
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
