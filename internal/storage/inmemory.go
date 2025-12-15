package storage

import (
	"context"
	"errors"

	"github.com/TPizik/url-shortener/internal/domain"
)

type InMemoryStorage struct {
	structure map[string]domain.ShortenedURL
}

func NewInMemoryStorage() (*InMemoryStorage, error) {
	storage := InMemoryStorage{
		structure: make(map[string]domain.ShortenedURL),
	}

	return &storage, nil
}

func (storage *InMemoryStorage) GetByShortURL(ctx context.Context, shortURL string) (*domain.ShortenedURL, error) {
	if url, ok := storage.structure[shortURL]; ok {
		return &url, nil
	}

	return nil, domain.ErrURLNotFound
}

func (storage *InMemoryStorage) GetURLsByUserID(ctx context.Context, userID string) ([]domain.ShortenedURL, error) {
	urls := make([]domain.ShortenedURL, 0)

	for _, value := range storage.structure {
		if value.UserID == userID {
			urls = append(urls, value)
		}
	}

	return urls, nil
}

func (storage *InMemoryStorage) FindByOriginalURL(ctx context.Context, originalURL string) (domain.ShortenedURL, error) {
	for _, value := range storage.structure {
		if value.OriginalURL == originalURL {
			return value, nil
		}
	}

	return domain.ShortenedURL{}, domain.ErrURLNotFound
}

func (storage *InMemoryStorage) SaveURL(ctx context.Context, dto domain.SaveShortURLDto) (*domain.ShortenedURL, error) {
	shortenedURL, err := storage.FindByOriginalURL(ctx, dto.OriginalURL)

	if err == nil {
		return &shortenedURL, domain.ErrURLConflict
	}

	storage.structure[dto.ShortURL] = domain.ShortenedURL{
		ID:          len(storage.structure) + 1,
		ShortURL:    dto.ShortURL,
		OriginalURL: dto.OriginalURL,
	}

	shortenedURL = storage.structure[dto.ShortURL]

	return &shortenedURL, nil
}

func (storage *InMemoryStorage) SaveSeveralURL(ctx context.Context, dtos []domain.SaveShortURLDto) ([]domain.ShortenedURL, error) {
	shortenedURLs := make([]domain.ShortenedURL, 0, len(dtos))

	for _, dto := range dtos {
		shortenedURL, err := storage.SaveURL(ctx, dto)

		if err != nil && !errors.Is(err, domain.ErrURLConflict) {
			return nil, err
		}

		shortenedURLs = append(shortenedURLs, *shortenedURL)
	}

	return shortenedURLs, nil
}

func (storage *InMemoryStorage) DeleteByShortURLs(ctx context.Context, shortURLs []string, userID string) error {
	for _, shortURL := range shortURLs {
		shortenedURL := storage.structure[shortURL]

		if shortenedURL.UserID != userID {
			continue
		}

		shortenedURL.IsDeleted = true
		storage.structure[shortURL] = shortenedURL
	}

	return nil
}

func (storage *InMemoryStorage) DoDeleteURLTasks(ctx context.Context, tasks []domain.DeleteURLsTask) error {
	for _, task := range tasks {
		for _, shortURL := range task.ShortURLs {
			shortenedURL := storage.structure[shortURL]

			if shortenedURL.UserID != task.UserID {
				continue
			}

			shortenedURL.IsDeleted = true
			storage.structure[shortURL] = shortenedURL
		}
	}

	return nil
}

func (storage *InMemoryStorage) Ping(ctx context.Context) error {
	return nil
}
