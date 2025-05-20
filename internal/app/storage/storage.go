package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"

	appErrors "github.com/TPizik/url-shortener/internal/app/errors"
)

type Storage struct {
	sync.RWMutex
	links map[string]string
}

func New() *Storage {
	return &Storage{
		links: map[string]string{},
	}
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
