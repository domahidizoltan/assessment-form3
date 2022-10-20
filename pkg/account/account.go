package account

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type (
	accountClient struct {
		client http.Client
	}
)

func NewClient() (accountClient, error) {
	return accountClient{
		client: *http.DefaultClient,
	}, nil
}

func (a accountClient) Fetch(accountID uuid.UUID) (*AccountData, error) {
	resp, err := a.client.Get("http://localhost:8080/v1/organisation/accounts/" + accountID.String())
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
