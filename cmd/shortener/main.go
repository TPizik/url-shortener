package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/TPizik/url-shortener/internal/app/config"
	"github.com/TPizik/url-shortener/internal/app/server"
	"github.com/TPizik/url-shortener/internal/app/services"
	"github.com/TPizik/url-shortener/internal/app/storage"
)

func main() {
	logger := services.InitLogger()
	defer logger.Sync()
	configVar := config.ParseConfig()
	storageVar, err := storage.NewStorage(context.Background(), &configVar)
	if err != nil {
		panic(err)
	}
	deleteURLQueue := services.NewDeleteURLQueue(storageVar, 2)
	go deleteURLQueue.Start(context.Background())
	defer storageVar.Close()
	serviceVar := services.NewService(storageVar, deleteURLQueue)
	serverVar := server.NewServer(serviceVar, configVar)
	go serverVar.ListenAndServe()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := serverVar.Shutdown(ctx); err != nil {
		panic("unexpected err on graceful shutdown")
	}
	logger.Infoln("main: done. exiting")
}
