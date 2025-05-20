package server

import (
	"context"
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
}

var sugar zap.SugaredLogger

func NewServer(service services.Service, config config.Config) Server {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar = *logger.Sugar()
	newServer := Server{service: service, srv: nil, config: config}

	r := chi.NewRouter()
	r.Post("/", (newServer.createRedirect))
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
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println("invalid parse body")
			return
		}
		url = strings.TrimSuffix(string(urlBytes), "\n")
	default:
		w.WriteHeader(http.StatusUnsupportedMediaType)
		fmt.Println("invalid ContentType")
		return
	}

	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("invalid url")
		return
	}

	fmt.Println("Add url", url)
	key, err := s.service.CreateRedirect(url)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err, key)
		return
	}
	resultURL := fmt.Sprintf("%s/%s", s.config.ShortAddr, key)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resultURL))
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("keyID")
	fmt.Println("Call redirect for", key)
	url, err := s.service.GetURLByKey(key)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(err, key)
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
