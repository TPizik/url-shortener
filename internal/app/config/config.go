package config

import (
	"flag"
	"os"
)

type Config struct {
	RunAddr         string
	ShortAddr       string
	FileStoragePath string
	DBDSN           string
}

func ParseConfig() Config {
	var flagRunAddr, flagShortAddr, flagStoragePath, flagDBDSN string

	flag.StringVar(&flagRunAddr, "a", "127.0.0.1:8080", "address and port to run server")
	flag.StringVar(&flagShortAddr, "b", "http://127.0.0.1:8080", "base address of the resulting shorthand url")
	flag.StringVar(&flagStoragePath, "f", "", "base path to storage file")
	flag.StringVar(&flagDBDSN, "d", "", "base path to database")
	flag.Parse()

	if envRunAddr := os.Getenv("SERVER_ADDRESS"); envRunAddr != "" {
		flagRunAddr = envRunAddr
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		flagShortAddr = envBaseURL
	}
	if envStoragePath := os.Getenv("FILE_STORAGE_PATH"); envStoragePath != "" {
		flagStoragePath = envStoragePath
	}
	if envDBDSN := os.Getenv("DATABASE_DSN"); envDBDSN != "" {
		flagDBDSN = envDBDSN
	}

	newConfig := Config{
		RunAddr:         flagRunAddr,
		ShortAddr:       flagShortAddr,
		FileStoragePath: flagStoragePath,
		DBDSN:           flagDBDSN,
	}
	return newConfig
}
