// Package account provides Form3 client to manage accounts.
// See https://www.api-docs.form3.tech/api/schemes/sepa-direct-debit/accounts/accounts
package account

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	conf "form3interview/internal/config"
	ire "form3interview/internal/requestenricher"
	"form3interview/pkg/config"
	re "form3interview/pkg/requestenricher"
)

const (
	accountsUrl  = "/organisation/accounts"
	accountsType = "accounts"
)

var (
	// ErrBaseUrlNotConfigured base url is not configured
	ErrBaseUrlNotConfigured = errors.New("baseUrl not configured")
	// ErrOrganisationIDNotConfigured organisation ID is not configured
	ErrOrganisationIDNotConfigured = errors.New("organisationID not configured")
	// ErrNilUUID nil UUID is not allowed
	ErrNilUUID = errors.New("nil UUID not allowed")
	// ErrAccountNotFound account not found
	ErrAccountNotFound = errors.New("account not found")
	// ErrInvalidAccountVersion account version not found
	ErrInvalidAccountVersion = errors.New("invalid account version")
	// ErrServerError server side error occured.
	// This includes these server errors:
	// 		500 Internal Server Error
	// 		502 Bad Gateway
	// 		504 Gateway Timeout
	ErrServerError = errors.New("server error")
	// ErrServerUnavailable server is unavailable
	ErrServerUnavailable = errors.New("server unavailable")
	// ErrUnexpectedServerResponse server response not handled by the client
	ErrUnexpectedServerResponse = errors.New("unexpected server response")
	// ErrInvalidRequest server returned with 400 Bad Request
	ErrInvalidRequest = errors.New("invalid request")

	generateUUID func() (uuid.UUID, error) = uuid.NewUUID
)

type (
	httpClient interface {
		Do(*http.Request, ...re.RequestEnricher) (*http.Response, error)
	}
	accountClient struct {
		client httpClient
		config conf.ClientConfig
	}
)

// NewClient creates a client for managing Form3 accounts.
// The client can be configured by passing config.Options with the helpers from the form3interview/pkg/config package.
func NewClient(options ...config.Option) (*accountClient, error) {
	cfg := conf.NewConfig()
	config.ApplyOptions(&cfg, options)

	if cfg.BaseUrl == nil || *cfg.BaseUrl == "" {
		return nil, ErrBaseUrlNotConfigured
	}

	if cfg.OrganisationID == nil || *cfg.OrganisationID == uuid.Nil {
		return nil, ErrOrganisationIDNotConfigured
	}

	return &accountClient{
		client: ire.EnrichClient(http.Client{
			Timeout:   *cfg.Timeout,
			Transport: createTransport(cfg),
		}),
		config: cfg,
	}, nil
}

// Create an account with attributes.
// See https://www.api-docs.form3.tech/api/schemes/sepa-direct-debit/accounts/accounts/create-an-account
//
// The request can be enriched by RequestEnricher
func (a accountClient) Create(attributes AccountAttributes, en ...re.RequestEnricher) (*AccountData, error) {
	newID, err := generateUUID()
	if err != nil {
		return nil, err
	}

	acc := AccountData{
		ID:             newID.String(),
		OrganisationID: a.config.OrganisationID.String(),
		Type:           accountsType,
		Attributes:     &attributes,
	}

	resp, err := a.post(acc, en...)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusBadRequest:
		msg, err := getErrorResponse(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Error().Msgf("%s: %s", ErrInvalidRequest, msg)
		return nil, ErrInvalidRequest
	case http.StatusInternalServerError, http.StatusGatewayTimeout, http.StatusBadGateway:
		msg, err := getErrorResponse(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Error().Msgf("%s: [%d] %s", ErrServerError, resp.StatusCode, msg)
		return nil, ErrServerError
	case http.StatusServiceUnavailable:
		return nil, ErrServerUnavailable
	case http.StatusCreated:
		log.Debug().Msgf("account %s created", acc.ID)
		return bodyToAccountData(resp.Body)
	}

	body := make([]byte, resp.ContentLength)
	if _, err := resp.Body.Read(body); err != nil {
		return nil, err
	}
	log.Info().Msgf("%s: [%d] %s", ErrUnexpectedServerResponse, resp.StatusCode, body)
	return nil, ErrUnexpectedServerResponse
}

// Fetch an account by it's ID
// See https://www.api-docs.form3.tech/api/schemes/sepa-direct-debit/accounts/accounts/fetch-an-account
//
// The request can be enriched by RequestEnricher
func (a accountClient) Fetch(accountID uuid.UUID, en ...re.RequestEnricher) (*AccountData, error) {
	if accountID == uuid.Nil {
		return nil, ErrNilUUID
	}

	resp, err := a.get(fmt.Sprintf("%s/%s", accountsUrl, accountID), en...)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, ErrAccountNotFound
	case http.StatusInternalServerError, http.StatusGatewayTimeout, http.StatusBadGateway:
		msg, err := getErrorResponse(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Error().Msgf("%s: [%d] %s", ErrServerError, resp.StatusCode, msg)
		return nil, ErrServerError
	case http.StatusServiceUnavailable:
		return nil, ErrServerUnavailable
	case http.StatusOK:
		return bodyToAccountData(resp.Body)
	}

	body := make([]byte, resp.ContentLength)
	if _, err := resp.Body.Read(body); err != nil {
		return nil, err
	}
	log.Info().Msgf("%s: [%d] %s", ErrUnexpectedServerResponse, resp.StatusCode, body)
	return nil, ErrUnexpectedServerResponse
}

// Delete is a convenience function to delete an account by it's ID having the latest version.
// See https://www.api-docs.form3.tech/api/schemes/sepa-direct-debit/accounts/accounts/delete-an-account
//
// Under the hood it fetches the latest account and delete that with the specific version returned.
// The request can be enriched by RequestEnricher
func (a accountClient) Delete(accountID uuid.UUID, en ...re.RequestEnricher) error {
	acc, err := a.Fetch(accountID, en...)
	if err != nil {
		return err
	}

	version := uint(0)
	if acc.Version != nil {
		version = uint(*acc.Version)
	}
	return a.DeleteVersion(accountID, version, en...)
}

// DeleteVersion deletes an account by it's ID having a specific version. 
// See https://www.api-docs.form3.tech/api/schemes/sepa-direct-debit/accounts/accounts/delete-an-account
//
// The request can be enriched by RequestEnricher
func (a accountClient) DeleteVersion(accountID uuid.UUID, version uint, en ...re.RequestEnricher) error {
	if accountID == uuid.Nil {
		return ErrNilUUID
	}

	url := fmt.Sprintf("%s/%s?version=%d", accountsUrl, accountID, version)
	resp, err := a.delete(url, en...)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrAccountNotFound
	case http.StatusConflict:
		msg, err := getErrorResponse(resp.Body)
		if err != nil {
			return err
		}
		log.Error().Msgf("%s: %s", ErrInvalidAccountVersion, msg)
		return ErrInvalidAccountVersion
	case http.StatusInternalServerError, http.StatusGatewayTimeout, http.StatusBadGateway:
		msg, err := getErrorResponse(resp.Body)
		if err != nil {
			return err
		}
		log.Error().Msgf("%s: [%d] %s", ErrServerError, resp.StatusCode, msg)
		return ErrServerError
	case http.StatusServiceUnavailable:
		return ErrServerUnavailable
	case http.StatusNoContent:
		log.Debug().Msgf("account %s deleted", accountID)
		return nil
	default:
		return err
	}
}

func (a accountClient) get(url string, en ...re.RequestEnricher) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, *a.config.BaseUrl+url, nil)
	if err != nil {
		return nil, err
	}
	return a.client.Do(req, en...)
}

func (a accountClient) post(account AccountData, en ...re.RequestEnricher) (*http.Response, error) {
	container := dataContainer{Data: account}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(container); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, *a.config.BaseUrl+accountsUrl, buf)
	if err != nil {
		return nil, err
	}
	return a.client.Do(req, en...)
}

func (a accountClient) delete(url string, en ...re.RequestEnricher) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, *a.config.BaseUrl+url, nil)
	if err != nil {
		return nil, err
	}
	return a.client.Do(req, en...)
}

func getErrorResponse(body io.ReadCloser) (string, error) {
	var se serverError
	if err := json.NewDecoder(body).Decode(&se); err != nil {
		if errors.Is(err, io.EOF) {
			return "", nil
		}
		return "", err
	}
	return se.ErrorMessage, nil
}

func toResponseBody(body string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(body))
}

func bodyToAccountData(body io.Reader) (*AccountData, error) {
	var container dataContainer
	if err := json.NewDecoder(body).Decode(&container); err != nil {
		return nil, err
	}
	return &container.Data, nil
}

func createTransport(cfg conf.ClientConfig) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = cfg.MaxConns
	transport.MaxIdleConnsPerHost = cfg.MaxConns
	transport.MaxIdleConns = cfg.MaxConns
	transport.IdleConnTimeout = *cfg.IdleConnTimeout
	return transport
}
