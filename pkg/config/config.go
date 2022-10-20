package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v6"
)

type ClientConfig struct {
	BaseUrl string         `env:"BASE_URL"`
	Timeout *time.Duration `env:"TIMEOUT" envDefault:"5s"`
}

func InitConfig() ClientConfig {
	cfg := ClientConfig{}
	if err := env.Parse(&cfg, env.Options{
		Prefix: "FORM3_",
	}); err != nil {
		log.Printf("failed to init config with env vars: %+v", err)
	}
	return cfg
}

type Option = func(*ClientConfig)

func WithBaseUrl(baseUrl string) Option {
	return func(c *ClientConfig) {
		c.BaseUrl = baseUrl
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *ClientConfig) {
		c.Timeout = &timeout
	}
}
