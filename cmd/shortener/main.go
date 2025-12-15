package main

import (
	"compress/gzip"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/TPizik/url-shortener/internal/config"
	httpHandlers "github.com/TPizik/url-shortener/internal/handlers/http"
	"github.com/TPizik/url-shortener/internal/logger"
	customMiddlewares "github.com/TPizik/url-shortener/internal/middlewares"
	"github.com/TPizik/url-shortener/internal/services"
	"github.com/TPizik/url-shortener/internal/storage"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env provided")
	}

	appConfig := &config.AppConfig{}
	appConfig.ParseFlags()

	customLogger, err := logger.NewLogger(logger.Options{
		Level:        logger.LogInfo,
		IsProduction: appConfig.AppEnvironment == config.AppProductionEnv,
	})
	if err != nil {
		panic(err)
	}

	gzipWriter, err := gzip.NewWriterLevel(nil, gzip.BestSpeed)
	if err != nil {
		panic(err)
	}

	urlStorage, err := storage.New(appConfig)
	if err != nil {
		panic(err)
	}

	stringGeneratorService := services.NewStringGenerator()
	userService := services.NewUserService()
	deleteURLQueue := services.NewDeleteURLQueue(urlStorage, customLogger, 3)
	shortenerService := services.NewShortenerService(
		urlStorage,
		stringGeneratorService,
		deleteURLQueue,
	)

	httpShortenerHandler := httpHandlers.NewShortenerHandler(
		appConfig,
		shortenerService,
	)

	httpRouter := makeRouter(
		httpShortenerHandler,
		userService,
		customLogger,
		gzipWriter,
	)

	workersCtx, workersStopCtx := context.WithCancel(context.Background())
	go deleteURLQueue.Start(workersCtx)

	log.Println("URL Shortener server is running on", appConfig.BaseHTTPAddr)
	log.Println("Config:", appConfig)

	httpServer := http.Server{
		Addr:    appConfig.BaseHTTPAddr,
		Handler: httpRouter,
	}

	go func() {
		err = httpServer.ListenAndServe()

		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGINT)
	<-sigs

	log.Println("start graceful shutdown...")

	shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer shutdownCtxCancel()

	go func() {
		<-shutdownCtx.Done()
		if shutdownCtx.Err() == context.DeadlineExceeded {
			log.Fatal("graceful shutdown timed out... forcing exit")
		}
	}()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}

	workersStopCtx()

	log.Println("graceful shutdown server successfully")
}

func makeRouter(
	shortenerHandler *httpHandlers.ShortenerHandler,
	userService *services.UserService,
	customLogger *logger.Logger,
	gzipWriter *gzip.Writer,
) http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.RealIP)
	mux.Use(middleware.Recoverer)
	mux.Use(customMiddlewares.NewCompressMiddleware(gzipWriter).Handler)
	mux.Use(func(handler http.Handler) http.Handler {
		return customMiddlewares.WithLogging(handler, customLogger)
	})
	mux.Use(func(handler http.Handler) http.Handler {
		return customMiddlewares.AuthMiddleware(handler, userService)
	})

	mux.Post("/api/shorten/batch", shortenerHandler.ShortBatchURL)
	mux.Post("/api/shorten", shortenerHandler.ShortURLJSON)
	mux.Post("/", shortenerHandler.ShortURL)
	mux.Delete("/api/user/urls", shortenerHandler.DeleteURLs)
	mux.Get("/api/user/urls", shortenerHandler.GetMyURLs)
	mux.Get("/ping", shortenerHandler.Ping)
	mux.Get("/{id}", shortenerHandler.RedirectToURLByID)

	return mux
}
