package config

import (
	"time"

	conf "form3interview/internal/config"

	"github.com/google/uuid"
)

type Option = func(*conf.ClientConfig)

func WithBaseUrl(baseUrl string) Option {
	return func(c *conf.ClientConfig) {
		c.BaseUrl = &baseUrl
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *conf.ClientConfig) {
		c.Timeout = &timeout
	}
}

func WithMaxConns(maxConn int) Option {
	return func(c *conf.ClientConfig) {
		c.MaxConns = maxConn
	}
}

func WithIdleConnTimeout(idleConnTimeout time.Duration) Option {
	return func(c *conf.ClientConfig) {
		c.IdleConnTimeout = &idleConnTimeout
	}
}

func WithOrganisationID(id uuid.UUID) Option {
	return func(c *conf.ClientConfig) {
		c.OrganisationID = &id
	}
}

func ApplyOptions(cfg *conf.ClientConfig, options []Option) {
	for _, opt := range options {
		opt(cfg)
	}
}
