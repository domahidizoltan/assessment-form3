package mocks

import (
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"
)

type HttpClientMock struct{ mock.Mock }

func (m *HttpClientMock) Get(url string) (*http.Response, error) {
	return getResponse(m.Called(url))
}

func (m *HttpClientMock) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	return getResponse(m.Called(url, contentType, body))
}

func (m *HttpClientMock) Do(req *http.Request) (*http.Response, error) {
	return getResponse(m.Called(req))
}

func getResponse(args mock.Arguments) (*http.Response, error) {
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}
