package mocks

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

type HttpClientMock struct{ mock.Mock }

func (m *HttpClientMock) Get(url string) (*http.Response, error) {
	args := m.Called(url)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func (m *HttpClientMock) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}
