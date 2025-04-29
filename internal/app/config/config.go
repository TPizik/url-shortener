package config

import (
	"flag"
)

type Config struct {
	RunAddr   string
	ShortAddr string
}

func ParseConfig() Config {
	var flagRunAddr, flagShortAddr string

	flag.StringVar(&flagRunAddr, "a", "127.0.0.1:8080", "address and port to run server")
	flag.StringVar(&flagShortAddr, "b", "http://127.0.0.1:8080", "base address of the resulting shorthand url")
	flag.Parse()

	newConfig := Config{
		RunAddr:   flagRunAddr,
		ShortAddr: flagShortAddr,
	}
	return newConfig
}
