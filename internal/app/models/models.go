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
