package models

type Redirect struct {
	URL string `json:"url"`
}

type ResultString struct {
	Result string `json:"result"`
}

type URLRowOriginal struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type URLRowShort struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type URLRow struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type ShortenedURL struct {
	Key         string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	IsDeleted   bool   `json:"is_deleted"`
}

type DeleteURLsTask struct {
	UserID    string
	ShortURLs []string
}
