package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/TPizik/url-shortener/internal/config"
	contextUtil "github.com/TPizik/url-shortener/internal/context"
	"github.com/TPizik/url-shortener/internal/domain"
	"github.com/TPizik/url-shortener/internal/handlers/http/dtos"
	httputils "github.com/TPizik/url-shortener/internal/handlers/http/http_utils"
	"github.com/go-chi/chi/v5"
)

type shortenerService interface {
	ShortURL(ctx context.Context, url string, userID string) (*domain.ShortenedURL, error)
	ShortBatchURL(ctx context.Context, urls []domain.ShortBatchURL, userID string) ([]domain.ShortBatchURL, error)
	GetUserURLs(ctx context.Context, userID string) ([]domain.ShortenedURL, error)
	DeleteURLs(ctx context.Context, urls []string, userID string) error
	GetByShortURL(ctx context.Context, url string) (*domain.ShortenedURL, error)
	Ping(ctx context.Context) error
}

type ShortenerHandler struct {
	config  *config.AppConfig
	service shortenerService
}

func NewShortenerHandler(
	config *config.AppConfig,
	service shortenerService,
) *ShortenerHandler {
	return &ShortenerHandler{
		config:  config,
		service: service,
	}
}

func (h *ShortenerHandler) ShortURLJSON(w http.ResponseWriter, r *http.Request) {
	userID, err := contextUtil.GetUserIDFromContext(r.Context())
	if err != nil {
		httputils.SendStatusCode(w, http.StatusUnauthorized)
		return
	}

	requestBody := dtos.ShortURLDto{}
	rawBody, err := io.ReadAll(r.Body)

	if err != nil {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	if jsonErr := json.Unmarshal(rawBody, &requestBody); jsonErr != nil {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	if requestBody.URL == "" {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	shortenedURL, err := h.service.ShortURL(r.Context(), requestBody.URL, userID)

	if errors.Is(err, domain.ErrURLConflict) {
		httputils.SendJSONResponse(w, http.StatusConflict, dtos.ShortURLResponse{
			Result: fmt.Sprintf("%s/%s", h.config.BaseShortURLAddr, shortenedURL.ShortURL),
		})
		return
	}

	if err != nil {
		httputils.SendStatusCode(w, http.StatusInternalServerError)
		return
	}

	httputils.SendJSONResponse(w, http.StatusCreated, dtos.ShortURLResponse{
		Result: fmt.Sprintf("%s/%s", h.config.BaseShortURLAddr, shortenedURL.ShortURL),
	})
}

func (h *ShortenerHandler) ShortBatchURL(w http.ResponseWriter, r *http.Request) {
	userID, err := contextUtil.GetUserIDFromContext(r.Context())
	if err != nil {
		httputils.SendStatusCode(w, http.StatusUnauthorized)
		return
	}

	requestBody := make([]dtos.ShortBatchURLDto, 0)
	rawBody, err := io.ReadAll(r.Body)

	if err != nil {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	if jsonErr := json.Unmarshal(rawBody, &requestBody); jsonErr != nil {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	if len(requestBody) == 0 {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	urls := make([]domain.ShortBatchURL, 0, len(requestBody))

	for _, url := range requestBody {
		urls = append(urls, domain.ShortBatchURL{
			OriginalURL:   url.OriginalURL,
			CorrelationID: url.CorrelationID,
		})
	}

	shortenedURLs, err := h.service.ShortBatchURL(r.Context(), urls, userID)

	if err != nil {
		log.Println(err)
		httputils.SendStatusCode(w, http.StatusInternalServerError)
		return
	}

	responseBody := make([]dtos.ShortBatchURLResponse, 0, len(shortenedURLs))

	for _, shortenedURL := range shortenedURLs {
		responseBody = append(responseBody, dtos.ShortBatchURLResponse{
			ShortURL:      fmt.Sprintf("%s/%s", h.config.BaseShortURLAddr, shortenedURL.ShortURL),
			CorrelationID: shortenedURL.CorrelationID,
		})
	}

	httputils.SendJSONResponse(w, 201, responseBody)
}

func (h *ShortenerHandler) ShortURL(w http.ResponseWriter, r *http.Request) {
	userID, err := contextUtil.GetUserIDFromContext(r.Context())
	if err != nil {
		httputils.SendStatusCode(w, http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)

	if err != nil {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	shortenedURL, err := h.service.ShortURL(r.Context(), string(body), userID)

	if errors.Is(err, domain.ErrURLConflict) {
		httputils.SendTextResponse(w, http.StatusConflict, fmt.Sprintf("%s/%s", h.config.BaseShortURLAddr, shortenedURL.ShortURL))
		return
	}

	if err != nil {
		httputils.SendStatusCode(w, http.StatusInternalServerError)
		return
	}

	httputils.SendTextResponse(w, http.StatusCreated, fmt.Sprintf("%s/%s", h.config.BaseShortURLAddr, shortenedURL.ShortURL))
}

func (h *ShortenerHandler) GetMyURLs(w http.ResponseWriter, r *http.Request) {
	userID, err := contextUtil.GetUserIDFromContext(r.Context())
	if err != nil {
		httputils.SendStatusCode(w, http.StatusUnauthorized)
		return
	}

	urls, err := h.service.GetUserURLs(r.Context(), userID)

	if err != nil {
		httputils.SendStatusCode(w, http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		httputils.SendStatusCode(w, http.StatusNoContent)
		return
	}

	responseURLs := make([]dtos.UserURLsResponse, 0, len(urls))

	for _, url := range urls {
		responseURLs = append(responseURLs, dtos.UserURLsResponse{
			ShortURL:    fmt.Sprintf("%s/%s", h.config.BaseShortURLAddr, url.ShortURL),
			OriginalURL: url.OriginalURL,
		})
	}

	httputils.SendJSONResponse(w, 200, responseURLs)
}

func (h *ShortenerHandler) DeleteURLs(w http.ResponseWriter, r *http.Request) {
	userID, err := contextUtil.GetUserIDFromContext(r.Context())
	if err != nil {
		httputils.SendStatusCode(w, http.StatusUnauthorized)
		return
	}

	var requestBody dtos.DeleteURLsRequest
	rawBody, err := io.ReadAll(r.Body)

	if err != nil {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(rawBody, &requestBody); err != nil {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	if len(requestBody) == 0 {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	h.service.DeleteURLs(r.Context(), requestBody, userID)

	httputils.SendStatusCode(w, http.StatusAccepted)
}

func (h *ShortenerHandler) RedirectToURLByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	originalURL, err := h.service.GetByShortURL(r.Context(), id)

	if err != nil {
		httputils.SendStatusCode(w, http.StatusBadRequest)
		return
	}

	if originalURL.IsDeleted {
		httputils.SendStatusCode(w, http.StatusGone)
		return
	}

	httputils.SendRedirectResponse(w, originalURL.OriginalURL)
}

func (h *ShortenerHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {
		httputils.SendStatusCode(w, http.StatusInternalServerError)
		return
	}

	httputils.SendStatusCode(w, http.StatusOK)
}
