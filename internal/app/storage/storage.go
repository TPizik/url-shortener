package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	appErrors "github.com/TPizik/url-shortener/internal/app/errors"
	"github.com/jmoiron/sqlx"
)

type PersistentStorageExpected interface {
	Load() (map[string]string, error)
	Add(key string, val string) error
}

type Storage struct {
	sync.RWMutex
	links   map[string]string
	storage PersistentStorageExpected
	db      *sqlx.DB
}

func New(persistent PersistentStorageExpected, db *sqlx.DB) (*Storage, error) {
	data, err := persistent.Load()
	if err != nil {
		return nil, err
	}
	return &Storage{
		links:   data,
		storage: persistent,
		db:      db,
	}, nil
}

func (c *Storage) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *Storage) Add(url string) (string, error) {
	c.Lock()
	defer c.Unlock()

	h := sha256.New()
	_, err := h.Write([]byte(url))
	if err != nil {
		return "", appErrors.ErrWrite
	}

	sha256Sum := h.Sum(nil)
	key := hex.EncodeToString(sha256Sum[:5])
	c.links[key] = url
	c.storage.Add(key, url)

	return key, nil
}

func (c *Storage) Get(key string) (string, error) {
	c.RLock()
	defer c.RUnlock()

	url, ok := c.links[key]
	if !ok {
		return "", appErrors.ErrKey
	}

	return url, nil
}
