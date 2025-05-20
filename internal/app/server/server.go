package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/TPizik/url-shortener/internal/app/config"
	"github.com/TPizik/url-shortener/internal/app/middleware"
	"github.com/TPizik/url-shortener/internal/app/services"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Server struct {
	service services.Service
	srv     *http.Server
	config  config.Config
	sugar   *zap.SugaredLogger
}

var sugar zap.SugaredLogger

func NewServer(service services.Service, config config.Config) Server {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar = *logger.Sugar()
	newServer := Server{service: service, srv: nil, config: config, sugar: &sugar}

	r := chi.NewRouter()
	r.Post("/", newServer.createRedirect)
	r.Post("/api/shorten", newServer.createRedirectJSON)
	r.Get("/{keyID}", newServer.redirect)

	srv := http.Server{
		Addr:    config.RunAddr,
		Handler: middleware.WithLogging(r, &sugar),
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
	default:
		s.error(w, http.StatusUnsupportedMediaType, "invalid ContentType")
		return
	}

	if url == "" {
		s.error(w, http.StatusBadRequest, "invalid url")
		return
	}

	key, err := s.service.CreateRedirect(url)
	if err != nil {
		s.error(w, http.StatusBadRequest, "invalid key")
		return
	}
	s.sugar.Infoln("Add url", url)
	resultURL := fmt.Sprintf("%s/%s", s.config.ShortAddr, key)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resultURL))
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("keyID")
	s.sugar.Infoln("Call redirect for", key)
	url, err := s.service.GetURLByKey(key)
	if err != nil {
		s.error(w, http.StatusBadRequest, "invalid key")
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) createRedirectJSON(w http.ResponseWriter, r *http.Request) {
	headerContentType := r.Header.Get("Content-Type")

	var redirect Redirect
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
	s.sugar.Infoln("Create redirect for", redirect.URL)
	key, err := s.service.CreateRedirect(redirect.URL)
	if err != nil {
		s.error(w, http.StatusBadRequest, "invalid key")
		return
	}
	result := ResultString{
		Result: fmt.Sprintf("%s/%s", s.config.ShortAddr, key),
	}

	response, _ := json.Marshal(result)
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(response))
}

func (s *Server) error(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	w.Header().Set("content-type", "plain/text")
	s.sugar.Infoln(msg)
	w.Write([]byte(msg))
}
