// Package config provides helper functions to configure Form3 clients.
package config

import (
	"time"

	conf "form3interview/internal/config"

	"github.com/google/uuid"
)

// Option is a function which will set the proper configuration field when a the client is created.
type Option = func(*conf.ClientConfig)

// WithBaseUrl will set the Form3 API base url.
// This will override the FORM3_BASE_URL env var.
func WithBaseUrl(baseUrl string) Option {
	return func(c *conf.ClientConfig) {
		c.BaseUrl = &baseUrl
	}
}

// WithTimeout will set the Form3 API client's global request timeout what is 5 seconds by default.
// This will override the FORM3_TIMEOUT env var.
func WithTimeout(timeout time.Duration) Option {
	return func(c *conf.ClientConfig) {
		c.Timeout = &timeout
	}
}

// WithMaxConns will set the Form3 API client's maximum number of connections what is 100 by default.
// This will override the FORM3_MAX_CONNS env var.
func WithMaxConns(maxConn int) Option {
	return func(c *conf.ClientConfig) {
		c.MaxConns = maxConn
	}
}

// WithIdleConnTimeout will set the Form3 API client's timeout for idle connections what is 90 seconds by default.
// This will override the FORM3_IDLE_CONN_TIMEOUT env var.
func WithIdleConnTimeout(idleConnTimeout time.Duration) Option {
	return func(c *conf.ClientConfig) {
		c.IdleConnTimeout = &idleConnTimeout
	}
}

// WithOrganisationID will set the organisation ID used by Form3 API calls.
// This will override the FORM3_ORGANISATION_ID env var.
func WithOrganisationID(id uuid.UUID) Option {
	return func(c *conf.ClientConfig) {
		c.OrganisationID = &id
	}
}

// ApplyOptions is used internally by the API clients to set option values on new clients.
func ApplyOptions(cfg *conf.ClientConfig, options []Option) {
	for _, opt := range options {
		opt(cfg)
	}
}
