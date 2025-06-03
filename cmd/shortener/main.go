package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/TPizik/url-shortener/internal/app/config"
	"github.com/TPizik/url-shortener/internal/app/server"
	"github.com/TPizik/url-shortener/internal/app/services"
	"github.com/TPizik/url-shortener/internal/app/storage"
)

func main() {
	configVar := config.ParseConfig()
	storageVar, err := storage.NewStorage(&configVar)
	if err != nil {
		panic(err)
	}
	defer storageVar.Close()
	serviceVar := services.NewService(storageVar)
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
	fmt.Println("main: done. exiting")
}
