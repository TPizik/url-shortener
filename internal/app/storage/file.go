package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sync"

	"github.com/TPizik/url-shortener/internal/app/config"
	"github.com/TPizik/url-shortener/internal/app/models"
)

const maxCapacity = 1024

type FileStorage struct {
	sync.RWMutex
	inmemory *InmemoryStorage
	file     *os.File
	filename string
	config   *config.Config
}

type RowFile struct {
	Key       string
	Value     string
	UserID    string
	IsDeleted bool
}

func NewFileStorage(filename string, config *config.Config) (*FileStorage, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}
	inmemory := NewInmemoryStorage(config)

	return &FileStorage{file: file, filename: filename, inmemory: inmemory, config: config}, nil
}

func (c *FileStorage) Close() error {
	return c.file.Close()
}

func (c *FileStorage) Drop() error {
	err := os.Remove(c.filename)
	if err != nil {
		return err
	}
	err = c.inmemory.Drop()
	if err != nil {
		return err
	}
	return nil
}

func (c *FileStorage) Ping(ctx context.Context) error {
	c.RLock()
	defer c.RUnlock()
	_, err := os.OpenFile(c.filename, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	return nil
}

func (c *FileStorage) Load() error {
	c.RLock()
	defer c.RUnlock()
	file, err := os.OpenFile(c.filename, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	data := make(map[string][]models.ShortenedURL)
	for scanner.Scan() {
		rawRow := scanner.Bytes()
		var row RowFile
		err := json.Unmarshal(rawRow, &row)
		if err != nil {
			return err
		}
		appendData := models.ShortenedURL{
			Key:         row.Key,
			OriginalURL: row.Value,
			IsDeleted:   row.IsDeleted,
		}
		data[row.UserID] = append(data[row.UserID], appendData)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := c.inmemory.Append(data); err != nil {
		return err
	}
	return nil
}

func (c *FileStorage) Add(ctx context.Context, url string, userID string) (string, error) {
	c.Lock()
	defer c.Unlock()
	key, err := c.inmemory.Add(ctx, url, userID)
	if err != nil {
		return "", err
	}
	row := RowFile{Key: key, Value: url, UserID: userID, IsDeleted: false}
	data, err := json.Marshal(row)
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	_, err = c.file.Write(data)

	if err != nil {
		return "", err
	}

	err = c.file.Sync()

	if err != nil {
		return "", err
	}

	return key, nil
}

func (c *FileStorage) Get(ctx context.Context, key string) (string, error) {
	c.RLock()
	defer c.RUnlock()
	url, err := c.inmemory.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (c *FileStorage) AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal, userID string) ([]models.URLRowShort, error) {
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

func (c *FileStorage) GetAllUserURLs(ctx context.Context, userID string) (map[string]string, error) {
	return c.inmemory.GetAllUserURLs(ctx, userID)
}

func (c *FileStorage) DoDeleteURLTasks(ctx context.Context, tasks []models.DeleteURLsTask) error {
	c.Lock()
	defer c.Unlock()
	for _, task := range tasks {
		userURLs, ok := c.inmemory.links[task.UserID]
		if !ok {
			return nil
		}
		for i, userURL := range userURLs {
			if slices.Contains(task.ShortURLs, userURL.Key) {
				c.inmemory.links[task.UserID][i].IsDeleted = true
			}
		}
	}
	b, err := json.Marshal(c.inmemory.links)
	if err != nil {
		return err
	}
	_, err = c.file.Write(b)
	if err != nil {
		return err
	}
	return nil
}
