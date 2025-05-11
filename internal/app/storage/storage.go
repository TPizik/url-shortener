package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
)

type StorageErrors struct {
	errKey   error
	errWrite error
}

type Storage struct {
	sync.RWMutex
	links map[string]string
	err   *StorageErrors
}

func New() *Storage {
	return &Storage{
		links: map[string]string{},
		err: &StorageErrors{
			errKey:   errors.New("key not exist"),
			errWrite: errors.New("error witch write key"),
		},
	}
}

func (c *Storage) Add(url string) (string, error) {
	c.Lock()
	defer c.Unlock()

	h := sha256.New()
	_, err := h.Write([]byte(url))
	if err != nil {
		return "", c.err.errWrite
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
		return "", c.err.errKey
	}

	return url, nil
}
