package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"sync"
)

const maxCapacity = 1024

type FileStorage struct {
	sync.RWMutex
	inmemory *InmemoryStorage
	file     *os.File
	filename string
}

type RowFile struct {
	Key   string
	Value string
}

func NewFileStorage(filename string) (*FileStorage, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}
	inmemory := NewInmemoryStorage()

	return &FileStorage{file: file, filename: filename, inmemory: inmemory}, nil
}

func (s *FileStorage) Close() error {
	return s.file.Close()
}

func (s *FileStorage) Ping(ctx context.Context) error {
	s.RLock()
	defer s.RUnlock()
	_, err := os.OpenFile(s.filename, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return err
	}
	return nil
}

func (s *FileStorage) Load() error {
	s.RLock()
	defer s.RUnlock()
	file, err := os.OpenFile(s.filename, os.O_RDONLY|os.O_CREATE, 0777)
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
	if err := s.inmemory.Append(data); err != nil {
		return err
	}
	return nil
}

func (s *FileStorage) Add(ctx context.Context, url string) (string, error) {
	s.Lock()
	defer s.Unlock()
	key, err := s.inmemory.Add(ctx, url)
	if err != nil {
		return "", err
	}
	row := RowFile{Key: key, Value: url}
	data, err := json.Marshal(row)
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	_, err = s.file.Write(data)

	if err != nil {
		return "", err
	}

	err = s.file.Sync()

	if err != nil {
		return "", err
	}

	return key, nil
}

func (s *FileStorage) Get(ctx context.Context, key string) (string, error) {
	s.RLock()
	defer s.RUnlock()
	url, err := s.inmemory.Get(ctx, key)
	if err != nil {
		return "", err
	}
	return url, nil
}
