package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	conf "form3interview/internal/config"
	"form3interview/pkg/config"
)

const (
	accountsUrl = "/organisation/accounts/"
)

var (
	ErrNilUuid                  = errors.New("accountID can't be nil UUID")
	ErrAccountNotFound          = errors.New("account not found")
	ErrInvalidAccountVersion    = errors.New("invalid account version")
	ErrServerError              = errors.New("server error")
	ErrUnexpectedServerResponse = errors.New("unexpected server response")
)

type (
	httpClient interface {
		Get(url string) (*http.Response, error)
		Do(req *http.Request) (*http.Response, error)
	}
	accountClient struct {
		client httpClient
		config conf.ClientConfig
	}
)

func NewClient(options ...config.Option) (accountClient, error) {
	cfg := conf.NewConfig()
	for _, opt := range options {
		opt(&cfg)
	}

	return accountClient{
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

func (a accountClient) Fetch(accountID uuid.UUID) (*AccountData, error) {
	if accountID == uuid.Nil {
		return nil, ErrNilUuid
	}

	resp, err := a.get(accountsUrl + accountID.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, ErrAccountNotFound
	case http.StatusInternalServerError:
		msg, err := getErrorResponse(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Error().Msgf("%s: %s", ErrServerError, msg)
		return nil, ErrServerError
	case http.StatusOK:
		var container responseContainer
		if err := json.NewDecoder(resp.Body).Decode(&container); err != nil {
			return nil, err
		}
		return &container.Data, nil
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
	case http.StatusInternalServerError:
		msg, err := getErrorResponse(resp.Body)
		if err != nil {
			return err
		}
		log.Error().Msgf("%s: %s", ErrServerError, msg)
		return ErrServerError
	case http.StatusNoContent:
		log.Debug().Msgf("account %s deleted", accountID)
		return nil
	default:
		return err
	}
}

func (a accountClient) get(url string) (*http.Response, error) {
	return a.client.Get(a.config.BaseUrl + url)
}

func (a accountClient) delete(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, a.config.BaseUrl+url, nil)
	if err != nil {
		return nil, err
	}
	return a.client.Do(req)
}

func getErrorResponse(body io.ReadCloser) (string, error) {
	var se serverError
	if err := json.NewDecoder(body).Decode(&se); err != nil {
		return "", err
	}
	return se.ErrorMessage, nil
}
