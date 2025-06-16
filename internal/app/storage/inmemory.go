package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/TPizik/url-shortener/internal/app/config"
	appErrors "github.com/TPizik/url-shortener/internal/app/errors"
	"github.com/TPizik/url-shortener/internal/app/models"
)

type InmemoryStorage struct {
	sync.RWMutex
	links  map[string]string
	config *config.Config
}

func NewInmemoryStorage(config *config.Config) *InmemoryStorage {
	links := make(map[string]string)
	return &InmemoryStorage{
		links:  links,
		config: config,
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

func (c *InmemoryStorage) AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal) ([]models.URLRowShort, error) {
	c.Lock()
	defer c.Unlock()
	shortURLs := make([]models.URLRowShort, 0)
	for _, url := range requestURLs {
		key, err := GetURLHash(url.OriginalURL)
		if err != nil {
			return nil, err
		}
		c.links[key] = url.OriginalURL
		shortURL := models.URLRowShort{
			CorrelationID: url.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", c.config.ShortAddr, key),
		}
		shortURLs = append(shortURLs, shortURL)
	}
	return shortURLs, nil
}
