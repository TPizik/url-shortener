package storage

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/TPizik/url-shortener/internal/app/config"
	appErrors "github.com/TPizik/url-shortener/internal/app/errors"
	"github.com/TPizik/url-shortener/internal/app/models"
)

type InmemoryStorage struct {
	sync.RWMutex
	links  map[string][]models.ShortenedURL
	config *config.Config
}

func NewInmemoryStorage(config *config.Config) *InmemoryStorage {
	links := make(map[string][]models.ShortenedURL)
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
	c.links = make(map[string][]models.ShortenedURL)
	return nil
}

func (c *InmemoryStorage) Append(data map[string][]models.ShortenedURL) error {
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
	userData, ok := c.links[userID]
	if ok {
		for _, data := range userData {
			if data.OriginalURL == url {
				return data.Key, appErrors.ErrConflict
			}
		}
	}

	dataURL := models.ShortenedURL{
		Key:         key,
		OriginalURL: url,
		IsDeleted:   false,
	}
	c.links[userID] = append(c.links[userID], dataURL)

	return key, nil
}

func (c *InmemoryStorage) Get(ctx context.Context, key string) (string, error) {
	var resURL string
	c.RLock()
	defer c.RUnlock()
	for _, userLinks := range c.links {
		for _, userLink := range userLinks {
			if key != userLink.Key {
				continue
			}
			if userLink.IsDeleted {
				return "", appErrors.ErrURLIsDeleted
			}
			resURL = userLink.OriginalURL
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
		data := models.ShortenedURL{
			Key:         key,
			OriginalURL: url.OriginalURL,
			IsDeleted:   false,
		}
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
		if userURLs.IsDeleted {
			continue
		}
		data[userURLs.Key] = userURLs.OriginalURL
	}
	return data, nil
}

func (c *InmemoryStorage) DoDeleteURLTasks(ctx context.Context, tasks []models.DeleteURLsTask) error {
	c.Lock()
	defer c.Unlock()
	for _, task := range tasks {
		userURLs, ok := c.links[task.UserID]
		if !ok {
			return nil
		}
		for i, userURL := range userURLs {
			if slices.Contains(task.ShortURLs, userURL.Key) {
				c.links[task.UserID][i].IsDeleted = true
			}
		}
	}
	return nil
}
