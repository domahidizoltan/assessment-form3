package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"form3interview/pkg/config"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
)

const (
	accountsUrl = "/organisation/accounts/"
)

var (
	ErrNilUuid               = errors.New("accountID can't be nil UUID")
	ErrAccountNotFound       = errors.New("account not found")
	ErrInvalidAccountVersion = errors.New("invalid account version")
)

type (
	httpClient interface {
		Get(url string) (resp *http.Response, err error)
		Do(req *http.Request) (*http.Response, error)
	}
	accountClient struct {
		client httpClient
		config config.ClientConfig
	}
)

func NewClient(options ...config.Option) (accountClient, error) {
	cfg := config.InitConfig()
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

func createTransport(cfg config.ClientConfig) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxConnsPerHost = cfg.MaxConns
	transport.MaxIdleConnsPerHost = cfg.MaxConns
	transport.MaxIdleConns = cfg.MaxConns
	transport.IdleConnTimeout = *cfg.IdleConnTimeout
	return transport
}

func (a accountClient) Fetch(accountID uuid.UUID) (*AccountData, error) {
	resp, err := a.get(accountsUrl + accountID.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var container responseContainer
	if err := json.NewDecoder(resp.Body).Decode(&container); err != nil {
		return nil, err
	}
	return &container.Data, nil

}

func (a accountClient) Delete(accountID uuid.UUID, version uint) error {
	if accountID == uuid.Nil {
		return ErrNilUuid
	}

	url := fmt.Sprintf("%s/%s?version=%d", accountsUrl, accountID, version)
	resp, err := a.delete(url)
	if err != nil {
		return err
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		return ErrAccountNotFound
	case http.StatusConflict:
		return ErrInvalidAccountVersion
	case http.StatusNoContent:
		log.Info().Msgf("account %s deleted", accountID)
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
