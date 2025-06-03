package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	Key   string
	Value string
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
	data := make(map[string]string)
	for scanner.Scan() {
		rawRow := scanner.Bytes()
		var row RowFile
		err := json.Unmarshal(rawRow, &row)
		if err == nil {
			data[row.Key] = row.Value
		}
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

func (c *FileStorage) Add(ctx context.Context, url string) (string, error) {
	c.Lock()
	defer c.Unlock()
	key, err := c.inmemory.Add(ctx, url)
	if err != nil {
		return "", err
	}
	row := RowFile{Key: key, Value: url}
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

func (c *FileStorage) AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal) ([]models.URLRowShort, error) {
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
