package config

import (
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type ClientConfig struct {
	OrganisationID  *uuid.UUID     `env:"ORGANISATION_ID"`
	BaseUrl         *string        `env:"BASE_URL"`
	Timeout         *time.Duration `env:"TIMEOUT" envDefault:"5s"`
	MaxConns        int            `env:"MAX_CONNS" envDefault:"100"`
	IdleConnTimeout *time.Duration `env:"IDLE_CONN_TIMEOUT" envDefault:"90s"`
}

func NewConfig() ClientConfig {
	cfg := ClientConfig{}
	if err := env.Parse(&cfg, env.Options{
		Prefix: "FORM3_",
	}); err != nil {
		log.Warn().Err(err).Msg("failed to init config with env vars")
	}
	return cfg
}
