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
	links  map[string][]map[string]string
	config *config.Config
}

func NewInmemoryStorage(config *config.Config) *InmemoryStorage {
	links := make(map[string][]map[string]string)
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

func (c *InmemoryStorage) Drop() error {
	for k := range c.links {
		delete(c.links, k)
	}
	return nil
}

func (c *InmemoryStorage) Append(data map[string][]map[string]string) error {
	c.Lock()
	defer c.Unlock()
	c.links = data
	return nil
}

func (c *InmemoryStorage) Add(ctx context.Context, url string, userID string) (string, error) {
	c.Lock()
	defer c.Unlock()

	key, err := GetURLHash(url)
	if err != nil {
		return "", err
	}
	urlData := map[string]string{key: url}
	c.links[userID] = append(c.links[userID], urlData)

	return key, nil
}

func (c *InmemoryStorage) Get(ctx context.Context, key string) (string, error) {
	var resURL string
	c.RLock()
	defer c.RUnlock()
	for _, userLinks := range c.links {
		for _, userLink := range userLinks {
			url, ok := userLink[key]
			if !ok {
				continue
			}
			resURL = url
		}
	}
	if resURL == "" {
		return "", appErrors.ErrKey
	}
	return resURL, nil
}

func (c *InmemoryStorage) AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal, userID string) ([]models.URLRowShort, error) {
	c.Lock()
	defer c.Unlock()
	shortURLs := make([]models.URLRowShort, 0)
	for _, url := range requestURLs {
		key, err := GetURLHash(url.OriginalURL)
		if err != nil {
			return nil, err
		}
		data := map[string]string{key: url.OriginalURL}
		c.links[userID] = append(c.links[userID], data)
		shortURL := models.URLRowShort{
			CorrelationID: url.CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", c.config.ShortAddr, key),
		}
		shortURLs = append(shortURLs, shortURL)
	}
	return shortURLs, nil
}

func (c *InmemoryStorage) GetAllUserURLs(ctx context.Context, userID string) (map[string]string, error) {
	var data = make(map[string]string)
	for _, userURLs := range c.links[userID] {
		for key, url := range userURLs {
			data[key] = url
		}
	}
	return data, nil
}
