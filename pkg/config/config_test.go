package config

import (
	"form3interview/internal/config"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	testBaseUrl        = "testhost"
	testOrganisationID = "ad27e265-9605-4b4b-a0e5-3003ea9cc4dc"
	orgIDKey           = "FORM3_ORGANISATION_ID"
	baseUrlKey         = "FORM3_BASE_URL"
	timeoutKey         = "FORM3_TIMEOUT"
	maxConnsKey        = "FORM3_MAX_CONNS"
	idleConnTimeoutKey = "FORM3_IDLE_CONN_TIMEOUT"
)

type configTestSuite struct {
	suite.Suite
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(configTestSuite))
}

func (s *configTestSuite) TestCreateFromEnvVars() {
	s.T().Setenv(orgIDKey, testOrganisationID)
	s.T().Setenv(baseUrlKey, testBaseUrl)
	s.T().Setenv(timeoutKey, "42s")
	s.T().Setenv(maxConnsKey, "42")
	s.T().Setenv(idleConnTimeoutKey, "42s")

	cfg := config.NewConfig()

	s.Equal(testOrganisationID, cfg.OrganisationID.String())
	s.Equal(testBaseUrl, *cfg.BaseUrl)
	s.Equal(42*time.Second, *cfg.Timeout)
	s.Equal(42, cfg.MaxConns)
	s.Equal(42*time.Second, *cfg.IdleConnTimeout)
}

func (s *configTestSuite) TestCreateWithDefaultValues() {
	cfg := config.NewConfig()

	s.Nil(cfg.OrganisationID)
	s.Nil(cfg.BaseUrl)
	s.Equal(5*time.Second, *cfg.Timeout)
	s.Equal(100, cfg.MaxConns)
	s.Equal(90*time.Second, *cfg.IdleConnTimeout)
}

func (s *configTestSuite) TestCreateWithOptions() {
	s.T().Setenv(orgIDKey, testOrganisationID)
	s.T().Setenv(baseUrlKey, testBaseUrl)
	s.T().Setenv(timeoutKey, "42s")
	s.T().Setenv(maxConnsKey, "42")
	s.T().Setenv(idleConnTimeoutKey, "42s")

	newOrgID := uuid.New()
	options := []Option{
		WithOrganisationID(newOrgID),
		WithBaseUrl("tst"),
		WithTimeout(2 * time.Second),
		WithMaxConns(2),
		WithIdleConnTimeout(2 * time.Second),
	}

	cfg := config.NewConfig()
	for _, opt := range options {
		opt(&cfg)
	}

	s.Equal(newOrgID, *cfg.OrganisationID)
	s.Equal("tst", *cfg.BaseUrl)
	s.Equal(2*time.Second, *cfg.Timeout)
	s.Equal(2, cfg.MaxConns)
	s.Equal(2*time.Second, *cfg.IdleConnTimeout)
}
