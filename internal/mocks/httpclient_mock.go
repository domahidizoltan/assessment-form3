package mocks

import (
	"form3interview/pkg/requestenricher"
	"net/http"

	"github.com/stretchr/testify/mock"
)

type HttpClientMock struct{ mock.Mock }

func (m *HttpClientMock) Do(req *http.Request, en ...requestenricher.RequestEnricher) (*http.Response, error) {
	args := m.Called(req, en)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}
