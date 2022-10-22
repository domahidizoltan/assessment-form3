package account

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	conf "form3interview/internal/config"
	"form3interview/pkg/config"
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
		Get(url string) (*http.Response, error)
		Post(url, contentType string, body io.Reader) (*http.Response, error)
		Do(req *http.Request) (*http.Response, error)
	}
	accountClient struct {
		client httpClient
		config conf.ClientConfig
	}
)

func NewClient(options ...config.Option) (*accountClient, error) {
	cfg := conf.NewConfig()
	for _, opt := range options {
		opt(&cfg)
	}

	if cfg.BaseUrl == nil {
		return nil, ErrBaseUrlNotConfigured
	}

	if cfg.OrganisationID == nil {
		return nil, ErrOrganisationIDNotConfigured
	}

	return &accountClient{
		client: &http.Client{
			Timeout:   *cfg.Timeout,
			Transport: createTransport(cfg),
		},
		config: cfg,
	}, nil
}

func createTransport(cfg conf.ClientConfig) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = cfg.MaxConns
	transport.MaxIdleConnsPerHost = cfg.MaxConns
	transport.MaxIdleConns = cfg.MaxConns
	transport.IdleConnTimeout = *cfg.IdleConnTimeout
	return transport
}

func (a accountClient) Create(attributes AccountAttributes) (*AccountData, error) {
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

	resp, err := a.post(acc)
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

func (a accountClient) Fetch(accountID uuid.UUID) (*AccountData, error) {
	if accountID == uuid.Nil {
		return nil, ErrNilUuid
	}

	resp, err := a.get(fmt.Sprintf("%s/%s", accountsUrl, accountID))
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

func (a accountClient) Delete(accountID uuid.UUID) error {
	acc, err := a.Fetch(accountID)
	if err != nil {
		return err
	}

	version := uint(0)
	if acc.Version != nil {
		version = uint(*acc.Version)
	}
	return a.DeleteVersion(accountID, version)
}

func (a accountClient) DeleteVersion(accountID uuid.UUID, version uint) error {
	if accountID == uuid.Nil {
		return ErrNilUuid
	}

	url := fmt.Sprintf("%s/%s?version=%d", accountsUrl, accountID, version)
	resp, err := a.delete(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrAccountNotFound
	case http.StatusConflict:
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

func (a accountClient) get(url string) (*http.Response, error) {
	return a.client.Get(*a.config.BaseUrl + url)
}

func (a accountClient) post(account AccountData) (*http.Response, error) {
	container := dataContainer{Data: account}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(container); err != nil {
		return &http.Response{Body: toResponseBody("")}, err
	}
	return a.client.Post(*a.config.BaseUrl+accountsUrl, jsonContentType, buf)
}

func (a accountClient) delete(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, *a.config.BaseUrl+url, nil)
	if err != nil {
		return &http.Response{Body: toResponseBody("")}, err
	}
	return a.client.Do(req)
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
	return ioutil.NopCloser(strings.NewReader(body))
}

func bodyToAccountData(body io.Reader) (*AccountData, error) {
	var container dataContainer
	if err := json.NewDecoder(body).Decode(&container); err != nil {
		return nil, err
	}
	return &container.Data, nil
}
