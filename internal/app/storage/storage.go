package storage

import (
	"errors"
	"strconv"
	"sync"
)

type Storage struct {
	sync.RWMutex
	links   map[string]string
	counter int
}

func New() *Storage {
	return &Storage{
		counter: 1,
		links:   map[string]string{},
	}
}

func (c *Storage) Add(url string) string {
	c.Lock()
	defer c.Unlock()

	c.counter++
	key := strconv.Itoa(c.counter)
	c.links[key] = url

	return key
}

func (c *Storage) Get(key string) (string, error) {
	c.RLock()
	defer c.RUnlock()

	url, ok := c.links[key]
	if !ok {
		return "", errors.New("key not exist")
	}

	return url, nil
}
