package services

import (
	"context"

	"github.com/TPizik/url-shortener/internal/app/models"
)

type IStorage interface {
	Get(ctx context.Context, key string) (string, error)
	Add(ctx context.Context, url string, userID string) (string, error)
	AddByBatch(ctx context.Context, requestURLs []models.URLRowOriginal, userID string) ([]models.URLRowShort, error)
	GetAllUserURLs(ctx context.Context, userID string) (map[string]string, error)
	Ping(ctx context.Context) error
	Close() error
	Drop() error
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

func (s *Service) Drop() error {
	return s.storage.Drop()
}

func (s *Service) CreateRedirect(ctx context.Context, key string, userID string) (string, error) {
	return s.storage.Add(ctx, key, userID)
}

func (s *Service) GetURLByKey(ctx context.Context, key string) (string, error) {
	return s.storage.Get(ctx, key)
}

func (s *Service) CreateRedirectByBatch(ctx context.Context, requestURLs []models.URLRowOriginal, userID string) ([]models.URLRowShort, error) {
	return s.storage.AddByBatch(ctx, requestURLs, userID)
}

func (s *Service) GetAllUserURLs(ctx context.Context, userID string) (map[string]string, error) {
	return s.storage.GetAllUserURLs(ctx, userID)
}

func (s *Service) Close() error {
	return s.storage.Close()
}
