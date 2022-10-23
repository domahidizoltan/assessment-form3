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
	accountsUrl     = "/organisation/accounts"
	jsonContentType = "application/json"
	accountsType    = "accounts"
)

var (
	ErrBaseUrlNotConfigured        = errors.New("baseUrl not configured")
	ErrOrganisationIDNotConfigured = errors.New("organisationID not configured")
	ErrNilUuid                     = errors.New("accountID can't be nil UUID")
	ErrAccountNotFound             = errors.New("account not found")
	ErrInvalidAccountVersion       = errors.New("invalid account version")
	ErrServerError                 = errors.New("server error")
	ErrServerUnavailable           = errors.New("server unavailable")
	ErrUnexpectedServerResponse    = errors.New("unexpected server response")
	ErrInvalidRequest              = errors.New("invalid request")

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

func (a accountClient) Fetch(accountID uuid.UUID, en ...re.RequestEnricher) (*AccountData, error) {
	if accountID == uuid.Nil {
		return nil, ErrNilUuid
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

func (a accountClient) DeleteVersion(accountID uuid.UUID, version uint, en ...re.RequestEnricher) error {
	if accountID == uuid.Nil {
		return ErrNilUuid
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
