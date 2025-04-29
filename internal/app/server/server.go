package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/TPizik/url-shortener/internal/app/services"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	service services.Service
	srv     *http.Server
}

const ServerAddr string = "127.0.0.1:8080"

func NewServer(service services.Service) Server {
	newServer := Server{service: service, srv: nil}

	r := chi.NewRouter()
	r.Post("/", newServer.createRedirect)
	r.Get("/{keyID}", newServer.redirect)

	srv := http.Server{
		Addr:    ServerAddr,
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
	key := s.service.CreateRedirect(url)
	resultURL := fmt.Sprintf("http://%s/%s", ServerAddr, key)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resultURL))
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("keyID")
	fmt.Println("Call redirect for", key)
	url, err := s.service.GetURLByKey(key)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("invalid key", key)
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
