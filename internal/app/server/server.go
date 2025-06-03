package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TPizik/url-shortener/internal/app/config"
	appErrors "github.com/TPizik/url-shortener/internal/app/errors"
	"github.com/TPizik/url-shortener/internal/app/models"
	"github.com/TPizik/url-shortener/internal/app/services"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Server struct {
	service     services.Service
	srv         *http.Server
	config      config.Config
	pingTimeout time.Duration
}

var Sugar zap.SugaredLogger

func NewServer(service services.Service, config config.Config) Server {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	Sugar = *logger.Sugar()
	newServer := Server{service: service, srv: nil, config: config, pingTimeout: 1 * time.Second}

	r := chi.NewRouter()
	r.Use(withLogging)
	r.Use(ungzipHandle)
	r.Use(gzipHandle)
	r.Post("/", newServer.createRedirect)
	r.Post("/api/shorten", newServer.createRedirectJSON)
	r.Post("/api/shorten/batch", newServer.createRedirectByBatch)
	r.Get("/{keyID}", newServer.redirect)
	r.Get("/ping", newServer.pingStorage)

	srv := http.Server{
		Addr:    config.RunAddr,
		Handler: r,
	}
	newServer.srv = &srv

	return newServer
}

func (s *Server) ListenAndServe() {
	s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func (s *Server) createRedirect(w http.ResponseWriter, r *http.Request) {
	headerContentType := r.Header.Get("Content-Type")
	w.Header().Set("content-type", "text/plain")
	var url string
	switch headerContentType {
	case "application/x-www-form-urlencoded":
		r.ParseForm()
		url = r.FormValue("url")
	case "text/plain; charset=utf-8":
		urlBytes, err := io.ReadAll(r.Body)
		if err != nil {
			s.error(w, http.StatusInternalServerError, "invalid parse body")
			return
		}
		url = strings.TrimSuffix(string(urlBytes), "\n")
	case "application/x-gzip":
		urlBytes, err := io.ReadAll(r.Body)
		if err != nil {
			s.error(w, http.StatusInternalServerError, "invalid body")
			return
		}
		url = strings.TrimSuffix(string(urlBytes), "\n")
	default:
		s.error(w, http.StatusUnsupportedMediaType, "invalid ContentType")
		return
	}

	if url == "" {
		s.error(w, http.StatusBadRequest, "invalid url")
		return
	}

	key, err := s.service.CreateRedirect(context.Background(), url)
	if err == appErrors.ErrConflict {
		Sugar.Infoln("Add url", url)
		resultURL := fmt.Sprintf("%s/%s", s.config.ShortAddr, key)
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(resultURL))
		return
	}
	if err != nil {
		s.error(w, http.StatusBadRequest, "invalid key")
		return
	}
	Sugar.Infoln("Add url", url)
	resultURL := fmt.Sprintf("%s/%s", s.config.ShortAddr, key)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resultURL))
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("keyID")
	Sugar.Infoln("Call redirect for", key)
	url, err := s.service.GetURLByKey(context.Background(), key)
	if err != nil {
		s.error(w, http.StatusBadRequest, "invalid key")
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) createRedirectJSON(w http.ResponseWriter, r *http.Request) {
	headerContentType := r.Header.Get("Content-Type")

	var redirect models.Redirect
	switch headerContentType {
	case "application/json":
		dataBytes, err := io.ReadAll(r.Body)
		if err != nil {
			s.error(w, http.StatusInternalServerError, "invalid parse body")
			return
		}
		err = json.Unmarshal(dataBytes, &redirect)
		if err != nil || redirect.URL == "" {
			s.error(w, http.StatusBadRequest, "invalid parse body")
			return
		}
	default:
		s.error(w, http.StatusUnsupportedMediaType, "invalid ContentType")
		return
	}
	Sugar.Infoln("Create redirect for", redirect.URL)
	key, err := s.service.CreateRedirect(context.Background(), redirect.URL)
	if err != nil {
		s.error(w, http.StatusBadRequest, "invalid key")
		return
	}
	result := models.ResultString{
		Result: fmt.Sprintf("%s/%s", s.config.ShortAddr, key),
	}

	response, _ := json.Marshal(result)
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(response))
}

func (s *Server) createRedirectByBatch(w http.ResponseWriter, r *http.Request) {
	headerContentType := r.Header.Get("Content-Type")
	if headerContentType != "application/json" {
		s.error(w, http.StatusUnsupportedMediaType, "invalid ContentType")
		return
	}
	dataBytes, err := io.ReadAll(r.Body)
	if err != nil {
		s.error(w, http.StatusInternalServerError, err.Error())
		return
	}
	requestURLs := make([]models.URLRowOriginal, 0)
	err = json.Unmarshal(dataBytes, &requestURLs)
	if err != nil {
		s.error(w, http.StatusBadRequest, "invalid parse body")
		return
	}

	responseURLs, err := s.service.CreateRedirectByBatch(context.Background(), requestURLs)
	if err != nil {
		s.error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response, err := json.Marshal(responseURLs)
	if err != nil {
		s.error(w, http.StatusInternalServerError, err.Error())
		return
	}
	status := http.StatusCreated
	if len(responseURLs) == 0 {
		status = http.StatusNoContent
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(response))
}

func (s *Server) pingStorage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(s.pingTimeout))
	defer cancel()
	err := s.service.Ping(ctx)
	if err != nil {
		s.error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) error(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Header().Set("content-type", "plain/text")
	Sugar.Infoln(msg)
	w.Write([]byte(msg))
}
