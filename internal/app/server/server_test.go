package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/TPizik/url-shortener/internal/app/config"
	"github.com/TPizik/url-shortener/internal/app/services"
	"github.com/TPizik/url-shortener/internal/app/storage"
	"github.com/bytedance/sonic"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/stretchr/testify/assert"
)

type TestServer struct {
	*httptest.Server
	service     services.Service
	config      config.Config
	pingTimeout time.Duration
}

func NewTestServer(t *testing.T) TestServer {
	config := config.Config{
		RunAddr:         "127.0.0.1:8080",
		ShortAddr:       "http://127.0.0.1:8080",
		FileStoragePath: "",
		DBDSN:           "",
	}
	storageTest, err := storage.NewStorage(context.Background(), &config)
	assert.Nil(t, err)
	deleteURLQueue := services.NewDeleteURLQueue(storageTest, 2)
	serviceTest := services.NewService(storageTest, deleteURLQueue)
	assert.Nil(t, err)
	s := NewServer(serviceTest, config)

	r := chi.NewRouter()
	r.Use(ungzipHandle)
	r.Use(gzipHandle)
	r.Use(setCookieHandler)
	r.Post("/", s.createRedirect)
	r.Post("/api/shorten/batch", s.createRedirectByBatch)
	r.Post("/api/shorten", s.createRedirectJSON)
	r.Get("/{keyID}", s.redirect)
	r.Get("/api/user/urls", s.getAllUserURLs)
	r.Delete("/api/user/urls", s.deleteURLs)
	ts := httptest.NewServer(r)

	srv := TestServer{service: serviceTest, Server: ts, config: config, pingTimeout: 1 * time.Second}

	return srv
}

func (s *TestServer) Close() {
	s.service.Drop()
	s.service.Close()
	s.Server.Close()
}

func TestServer_createRedirect(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()
	jar, _ := cookiejar.New(nil)
	client := http.Client{Jar: jar}
	reqURL := fmt.Sprintf("%s/", ts.URL)

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
			name:        "positive test conflict",
			method:      http.MethodPost,
			contentType: "application/x-www-form-urlencoded",
			code:        409,
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
			method:      http.MethodPatch,
			contentType: "application/x-www-form-urlencoded",
			code:        405,
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
			data := url.Values{}
			data.Set(tt.urlKey, tt.urlVal)

			request, err := http.NewRequest(tt.method, reqURL, bytes.NewBufferString(data.Encode()))
			assert.Nil(t, err)

			request.Header.Set("Content-Type", tt.contentType)
			res, err := client.Do(request)

			assert.Nil(t, err)
			assert.Equal(t, tt.code, res.StatusCode, "statuses should be equal")

			defer res.Body.Close()
		})
	}
}

func TestServer_redirect(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()
	location := "http://example-test.com/..."
	jar, err := cookiejar.New(nil)
	assert.Nil(t, err)
	client := http.Client{
		Jar: jar,
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	data := url.Values{}
	data.Set("url", location)
	request, err := http.NewRequest(http.MethodPost, ts.URL, bytes.NewBufferString(data.Encode()))
	assert.Nil(t, err)

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(request)
	assert.Nil(t, err)

	dataBytes, err := io.ReadAll(res.Body)
	assert.Nil(t, err)
	sepResData := strings.Split(string(dataBytes), "/")
	validKey := sepResData[len(sepResData)-1]

	defer res.Body.Close()
	res, err = client.Do(request)
	assert.Nil(t, err)
	defer res.Body.Close()
	tests := []struct {
		name     string
		method   string
		url      string
		code     int
		location string
	}{
		{
			name:     "positive test1",
			method:   http.MethodGet,
			url:      fmt.Sprintf("/%s", validKey),
			code:     307,
			location: location,
		},
		{
			name:     "negative test2",
			method:   http.MethodGet,
			url:      "/invalid",
			code:     400,
			location: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("%s%s", ts.URL, tt.url)
			res, err := client.Get(url)

			assert.Nil(t, err)
			defer res.Body.Close()

			assert.Equal(t, res.StatusCode, tt.code, "statuses should be equal")

			if tt.code == 307 {
				loc := res.Header.Get("location")
				assert.Equal(t, loc, tt.location, "statuses should be equal")
			}

		})
	}
}

func TestServer_createRedirectJSON(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	client := http.Client{}
	url := fmt.Sprintf("%s/api/shorten", ts.URL)
	key, err := storage.GetURLHash(url)
	validResponse := fmt.Sprintf("%s/%s", ts.URL, key)
	assert.Nil(t, err)
	type request struct {
		URL string `json:"url"`
	}

	type response struct {
		Result string `json:"result"`
	}

	tests := []struct {
		name        string
		method      string
		contentType string
		data        request
		code        int
		result      response
	}{
		{
			name:        "positive test1",
			method:      http.MethodPost,
			contentType: "application/json",
			data:        request{URL: "http://example-test-json.com"},
			code:        201,
			result:      response{Result: validResponse},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := sonic.Marshal(tt.data)
			assert.Nil(t, err)

			req, _ := http.NewRequest(tt.method, url, bytes.NewBuffer(data))
			req.Header.Set("Content-Type", tt.contentType)
			res, err := client.Do(req)
			assert.Nil(t, err)

			assert.Equal(t, res.StatusCode, tt.code, "statuses should be equal")

			defer res.Body.Close()
			if tt.code == 201 {
				bodyBytes, err := io.ReadAll(res.Body)
				assert.Nil(t, err)
				body := response{}
				assert.Nil(t, sonic.Unmarshal(bodyBytes, &body))
			}
		})
	}
}

func TestServer_GetAllUserURLs(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	jar, _ := cookiejar.New(nil)

	client := http.Client{Jar: jar}
	assert := assert.New(t)

	type row struct {
		ShortURL    string `json:"short_url"`
		OriginalURL string `json:"original_url"`
	}
	type response struct {
		ShortURL string `json:"result"`
	}
	type request struct {
		URL string `json:"url"`
	}

	expected := []row{
		{
			ShortURL:    "http://127.0.0.1:8080/6287ba30f5",
			OriginalURL: "http://example.com/1",
		},
		{
			ShortURL:    "http://127.0.0.1:8080/a135beced0",
			OriginalURL: "http://example.com/2",
		},
	}
	// get empty list
	resp, err := client.Get(fmt.Sprintf("%s/api/user/urls", ts.URL))
	assert.Equal(http.StatusNoContent, resp.StatusCode, "invalid status")
	assert.Nil(err)

	bodyBytes, err := io.ReadAll(resp.Body)
	assert.Nil(err)
	defer resp.Body.Close()
	body := make([]row, 0)
	sonic.Unmarshal(bodyBytes, &body)
	assert.Equal(body, make([]row, 0), "body should be empty")

	for i := range expected {
		contentType := "application/json"
		url := fmt.Sprintf("%s/api/shorten", ts.URL)
		data := request{URL: expected[i].OriginalURL}
		dataByte, err := sonic.Marshal(data)
		assert.Nil(err)
		resp, err := client.Post(url, contentType, bytes.NewBuffer(dataByte))
		assert.Nil(err)
		assert.Equal(resp.StatusCode, 201, "statuses should be equal")
		bodyBytes, err = io.ReadAll(resp.Body)
		assert.Nil(err)
		defer resp.Body.Close()
		var body response
		err = sonic.Unmarshal(bodyBytes, &body)
		assert.Nil(err)
		expected[i].ShortURL = body.ShortURL
	}

	// get list
	resp, err = client.Get(fmt.Sprintf("%s/api/user/urls", ts.URL))
	assert.Nil(err)
	assert.Equal(http.StatusOK, resp.StatusCode, "invalid status")

	bodyBytes, err = io.ReadAll(resp.Body)
	assert.Nil(err)
	defer resp.Body.Close()
	body = make([]row, 0)
	sonic.Unmarshal(bodyBytes, &body)
	assert.Equal(body, expected, "body is wrong. Got %v, want %v", body, expected)
}

func TestServer_DeleteUrls(t *testing.T) {
	ts := NewTestServer(t)
	defer ts.Close()

	jar, _ := cookiejar.New(nil)
	client := http.Client{Jar: jar}
	creaateUrl := fmt.Sprintf("%s/api/shorten", ts.URL)
	deleteUrl := fmt.Sprintf("%s/api/user/urls", ts.URL)

	type createRequest struct {
		URL string `json:"url"`
	}
	type createResponse struct {
		Result string `json:"result"`
	}

	tests := []struct {
		name        string
		method      string
		contentType string
		code        int
	}{
		{
			name:        "positive test1",
			method:      http.MethodDelete,
			contentType: "text/plain",
			code:        202,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestContentType := "application/json"
			data := createRequest{URL: "http://example.com"}
			requestData, err := sonic.Marshal(data)
			assert.Nil(t, err)

			req, _ := http.NewRequest(http.MethodPost, creaateUrl, bytes.NewBuffer(requestData))
			req.Header.Set("Content-Type", requestContentType)
			res, err := client.Do(req)
			assert.Nil(t, err)

			assert.Equal(t, http.StatusCreated, res.StatusCode, "statuses should be equal")

			defer res.Body.Close()
			bodyBytes, err := io.ReadAll(res.Body)
			assert.Nil(t, err)
			body := createResponse{}
			assert.Nil(t, sonic.Unmarshal(bodyBytes, &body))
			key := []string{body.Result}
			deleteRequestData, err := sonic.Marshal(key)
			assert.Nil(t, err)
			reqDelete, _ := http.NewRequest(http.MethodDelete, deleteUrl, bytes.NewBuffer(deleteRequestData))
			reqDelete.Header.Set("Content-Type", requestContentType)
			resDelete, err := client.Do(reqDelete)
			assert.Nil(t, err)

			assert.Equal(t, http.StatusAccepted, resDelete.StatusCode, "statuses should be equal")

		})
	}
}
