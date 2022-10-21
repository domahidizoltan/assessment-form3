package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v6"
)

type ClientConfig struct {
	BaseUrl         string         `env:"BASE_URL"`
	Timeout         *time.Duration `env:"TIMEOUT" envDefault:"5s"`
	MaxConns        int            `env:"MAX_CONNS" envDefault:"100"`
	IdleConnTimeout *time.Duration `env:"IDLE_CONN_TIMEOUT" envDefault:"90s"`
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

func WithMaxConns(maxConn int) Option {
	return func(c *ClientConfig) {
		c.MaxConns = maxConn
	}
}

func WithIdleConnTimeout(idleConnTimeout time.Duration) Option {
	return func(c *ClientConfig) {
		c.IdleConnTimeout = &idleConnTimeout
	}
}
