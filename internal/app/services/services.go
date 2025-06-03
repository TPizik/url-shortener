package services

import (
	"context"

	"github.com/TPizik/url-shortener/internal/app/models"
)

type IStorage interface {
	Get(ctx context.Context, key string) (string, error)
	Add(ctx context.Context, url string) (string, error)
	AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal) ([]models.URLRowShort, error)
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

func (s *Service) CreateRedirect(ctx context.Context, key string) (string, error) {
	return s.storage.Add(ctx, key)
}

func (s *Service) GetURLByKey(ctx context.Context, key string) (string, error) {
	return s.storage.Get(ctx, key)
}

func (s *Service) CreateRedirectByBatch(ctx context.Context, requestURLs []models.URLRowOriginal) ([]models.URLRowShort, error) {
	return s.storage.AddByBatch(ctx, requestURLs)
}
