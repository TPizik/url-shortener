package storage

import (
	"bufio"
	"encoding/json"
	"os"
)

const maxCapacity = 1024

type FileStorage struct {
	file     *os.File
	filename string
}

type Row struct {
	Key   string
	Value string
}

func NewFileStorage(filename string) (*FileStorage, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}

	return &FileStorage{file: file, filename: filename}, nil
}

func (s *FileStorage) Close() error {
	return s.file.Close()
}

func (s *FileStorage) Load() (map[string]string, error) {
	file, err := os.OpenFile(s.filename, os.O_RDONLY|os.O_CREATE, 0777)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(file)
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)
	data := make(map[string]string)
	for scanner.Scan() {
		rawRow := scanner.Bytes()
		var row Row
		err := json.Unmarshal(rawRow, &row)
		if err == nil {
			data[row.Key] = row.Value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *FileStorage) Add(key string, val string) error {
	row := Row{Key: key, Value: val}
	data, err := json.Marshal(row)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = s.file.Write(data)

	if err != nil {
		return err
	}

	err = s.file.Sync()

	if err != nil {
		return err
	}

	return nil
}
