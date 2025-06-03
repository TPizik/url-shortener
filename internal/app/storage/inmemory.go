package storage

import (
	"context"
	"sync"

	appErrors "github.com/TPizik/url-shortener/internal/app/errors"
)

type InmemoryStorage struct {
	sync.RWMutex
	links map[string]string
}

func NewInmemoryStorage() *InmemoryStorage {
	links := make(map[string]string)
	return &InmemoryStorage{
		links: links,
	}
}

func (c *InmemoryStorage) Ping(ctx context.Context) error {
	return nil
}

func (c *InmemoryStorage) Close() error {
	return nil
}

func (c *InmemoryStorage) Append(data map[string]string) error {
	c.Lock()
	defer c.Unlock()
	c.links = data
	return nil
}

func (c *InmemoryStorage) Add(ctx context.Context, url string) (string, error) {
	c.Lock()
	defer c.Unlock()

	key, err := GetURLHash(url)
	if err != nil {
		return "", err
	}
	c.links[key] = url

	return key, nil
}

func (c *InmemoryStorage) Get(ctx context.Context, key string) (string, error) {
	c.RLock()
	defer c.RUnlock()

	url, ok := c.links[key]
	if !ok {
		return "", appErrors.ErrKey
	}

	return url, nil
}
