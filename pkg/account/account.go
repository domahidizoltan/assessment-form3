package account

import (
	"encoding/json"
	"form3interview/pkg/config"
	"net/http"

	"github.com/google/uuid"
)

type (
	accountClient struct {
		client http.Client
		config config.ClientConfig
	}
)

func NewClient(options ...config.Option) (accountClient, error) {
	cfg := config.InitConfig()
	for _, opt := range options {
		opt(&cfg)
	}

	return accountClient{
		client: *&http.Client{
			Timeout: *cfg.Timeout,
		},
		config: cfg,
	}, nil
}

func (a accountClient) Fetch(accountID uuid.UUID) (*AccountData, error) {
	resp, err := a.get("/organisation/accounts/" + accountID.String())
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

func (a accountClient) get(url string) (*http.Response, error) {
	return a.client.Get(a.config.BaseUrl + url)
}
