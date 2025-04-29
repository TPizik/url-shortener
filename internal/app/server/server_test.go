package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/TPizik/url-shortener/internal/app/services"
	"github.com/TPizik/url-shortener/internal/app/storage"
	"github.com/go-chi/chi/v5"
)

func TestServer_createRedirect(t *testing.T) {
	var storageTest = storage.New()
	var serviceTest = services.NewService(storageTest)
	tests := []struct {
		name        string
		method      string
		contentType string
		code        int
		urlKey      string
		urlVal      string
	}{
		{
			name:        "positive test1",
			method:      http.MethodPost,
			contentType: "application/x-www-form-urlencoded",
			code:        201,
			urlKey:      "url",
			urlVal:      "http://example.com/...",
		},
		{
			name:        "negative data",
			method:      http.MethodPost,
			contentType: "application/x-www-form-urlencoded",
			code:        400,
			urlKey:      "url0",
			urlVal:      "http://example.com/...",
		},
		{
			name:        "negative empty url",
			method:      http.MethodPost,
			contentType: "application/x-www-form-urlencoded",
			code:        400,
			urlKey:      "url0",
			urlVal:      "",
		},
		{
			name:        "negative invalid method",
			method:      http.MethodGet,
			contentType: "application/x-www-form-urlencoded",
			code:        400,
			urlKey:      "url",
			urlVal:      "http://example.com/...",
		},
		{
			name:        "negative invalid content type",
			method:      http.MethodPost,
			contentType: "application/json",
			code:        415,
			urlKey:      "url",
			urlVal:      "http://example.com/...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(serviceTest)
			data := url.Values{}
			data.Set(tt.urlKey, tt.urlVal)

			request := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(data.Encode()))
			request.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			h := http.HandlerFunc(s.createRedirect)

			h.ServeHTTP(w, request)
			res := w.Result()
			if res.StatusCode != tt.code {
				t.Errorf("Expected status code %d, got %d", tt.code, w.Code)
			}
			defer res.Body.Close()
		})
	}
}

func TestServer_redirect(t *testing.T) {
	var storageTest = storage.New()
	var serviceTest = services.NewService(storageTest)
	var location = "https://example.com"
	var validKey = serviceTest.CreateRedirect(location)
	client := http.Client{}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	tests := []struct {
		name     string
		method   string
		code     int
		url      string
		location string
	}{
		{
			name:     "positive test1",
			method:   http.MethodGet,
			code:     307,
			url:      fmt.Sprintf("/%s", validKey),
			location: location,
		},
		{
			name:     "negative test2",
			method:   http.MethodGet,
			code:     400,
			url:      "/invalid",
			location: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(serviceTest)

			r := chi.NewRouter()
			r.Get("/{keyID}", s.redirect)
			ts := httptest.NewServer(r)
			defer ts.Close()
			url := fmt.Sprintf("%s%s", ts.URL, tt.url)
			fmt.Println("Url - ", url)
			res, err := client.Get(url)
			if err != nil {
				t.Errorf("Problem with server")
			}
			defer res.Body.Close()
			if res.StatusCode != tt.code {
				t.Errorf("Expected status code %d, got %d", tt.code, res.StatusCode)
			}
			if tt.code == 307 {
				loc := res.Header.Get("location")
				if loc != tt.location {
					t.Errorf("Expected location %s, got %s", tt.location, loc)
				}
			}

		})
	}
}
