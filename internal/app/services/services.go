package services

import "context"

type IStorage interface {
	Get(key string) (string, error)
	Add(url string) (string, error)
	Ping(ctx context.Context) error
}

type Service struct {
	storage IStorage
}

func NewService(storage IStorage) Service {
	return Service{
		storage: storage,
	}
}

func (s *Service) Ping(ctx context.Context) error {
	return s.storage.Ping(ctx)
}

func (s *Service) CreateRedirect(key string) (string, error) {
	return s.storage.Add(key)
}

func (s *Service) GetURLByKey(key string) (string, error) {
	return s.storage.Get(key)
}
