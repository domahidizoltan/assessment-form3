package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"form3interview/internal/config"
	"form3interview/internal/mocks"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	Do                 = "Do"
	testBaseUrl        = "testhost"
	testAccountsUrl    = testBaseUrl + accountsUrl
	testOrganisationID = "ad27e265-9605-4b4b-a0e5-3003ea9cc4dc"
)

type accountTestSuite struct {
	suite.Suite
	mockHttpClient *mocks.HttpClientMock
	accountClient  accountClient
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(accountTestSuite))
}

func (s *accountTestSuite) SetupTest() {
	s.mockHttpClient = &mocks.HttpClientMock{}
	orgID := uuid.MustParse(testOrganisationID)
	baseUrl := testBaseUrl
	s.accountClient = accountClient{
		client: s.mockHttpClient,
		config: config.ClientConfig{
			BaseUrl:        &baseUrl,
			OrganisationID: &orgID,
		},
	}
}

func (s *accountTestSuite) TestCreateReturnsError() {
	for _, test := range []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedError  error
	}{
		{
			name:           "invalid request",
			responseStatus: http.StatusBadRequest,
			responseBody:   "{\"error_message\":\"base_currency is required\"}",
			expectedError:  ErrInvalidRequest,
		},
		{
			name:           "server error",
			responseStatus: http.StatusInternalServerError,
			responseBody:   "{\"error_message\": \"backend error\"}",
			expectedError:  ErrServerError,
		},
		{
			name:           "server bad gateway",
			responseStatus: http.StatusBadGateway,
			expectedError:  ErrServerError,
		},
		{
			name:           "server gateway timeout",
			responseStatus: http.StatusGatewayTimeout,
			expectedError:  ErrServerError,
		},
		{
			name:           "server unavailable",
			responseStatus: http.StatusServiceUnavailable,
			expectedError:  ErrServerUnavailable,
		},
		{
			name:           "unexpected server response",
			responseStatus: http.StatusTeapot,
			responseBody:   "oops",
			expectedError:  ErrUnexpectedServerResponse,
		},
	} {
		length := int64(len(test.responseBody))
		s.Run(test.name, func() {
			s.mockHttpClient.
				On(Do, mock.MatchedBy(postRequestMatcher(AccountData{})), mock.Anything).
				Return(&http.Response{Body: toResponseBody(test.responseBody), StatusCode: test.responseStatus, ContentLength: length}, nil).
				Once()

			_, actualErr := s.accountClient.Create(AccountAttributes{})
			s.ErrorIs(test.expectedError, actualErr)
		})
	}
}

func (s *accountTestSuite) TestCreateReturnsHttpClientError() {
	expectedError := errors.New("http client error")
	s.mockHttpClient.
		On(Do, mock.MatchedBy(postRequestMatcher(AccountData{})), mock.Anything).
		Return(nil, expectedError).
		Once()

	_, actualError := s.accountClient.Create(AccountAttributes{})

	s.ErrorIs(expectedError, actualError)
}

func (s *accountTestSuite) TestCreateAccount() {
	accountID := uuid.New()
	originalGenerateUUID := generateUUID
	generateUUID = func() (uuid.UUID, error) { return accountID, nil }
	defer func() {
		generateUUID = originalGenerateUUID
	}()

	atr := AccountAttributes{BaseCurrency: "EUR"}

	fakeResponse := "{\"data\":{}}"
	s.mockHttpClient.
		On(Do, mock.MatchedBy(postRequestMatcher(AccountData{})), mock.Anything).
		Return(&http.Response{Body: toResponseBody(fakeResponse), StatusCode: http.StatusCreated}, nil).
		Once()

	_, err := s.accountClient.Create(atr)
	s.NoError(err)
	request := s.mockHttpClient.Calls[0].Arguments[0].(*http.Request)
	requestedAccount, err := bodyToAccountData(request.Body)
	s.Require().NoError(err)
	s.Equal(accountID.String(), requestedAccount.ID)
	s.Equal(testOrganisationID, requestedAccount.OrganisationID)
	s.Equal(accountsType, requestedAccount.Type)
	s.Equal("EUR", requestedAccount.Attributes.BaseCurrency)
}

func (s *accountTestSuite) TestFetchReturnsError_WhenNilUuidGiven() {
	_, actualError := s.accountClient.Fetch(uuid.Nil)

	s.ErrorIs(ErrNilUUID, actualError)
	s.mockHttpClient.AssertNotCalled(s.T(), Do)
}

func (s *accountTestSuite) TestFetchReturnsError() {
	for _, test := range []struct {
		name           string
		accountID      uuid.UUID
		responseStatus int
		responseBody   string
		expectedError  error
	}{
		{
			name:           "account not found",
			accountID:      uuid.New(),
			responseStatus: http.StatusNotFound,
			expectedError:  ErrAccountNotFound,
		}, {
			name:           "server error",
			accountID:      uuid.New(),
			responseStatus: http.StatusInternalServerError,
			responseBody:   "{\"error_message\": \"backend error\"}",
			expectedError:  ErrServerError,
		},
		{
			name:           "server bad gateway",
			accountID:      uuid.New(),
			responseStatus: http.StatusBadGateway,
			expectedError:  ErrServerError,
		},
		{
			name:           "server gateway timeout",
			accountID:      uuid.New(),
			responseStatus: http.StatusGatewayTimeout,
			expectedError:  ErrServerError,
		},
		{
			name:           "server unavailable",
			accountID:      uuid.New(),
			responseStatus: http.StatusServiceUnavailable,
			expectedError:  ErrServerUnavailable,
		},
		{
			name:           "unexpected server response",
			accountID:      uuid.New(),
			responseStatus: http.StatusTeapot,
			responseBody:   "oops",
			expectedError:  ErrUnexpectedServerResponse,
		},
	} {
		s.Run(test.name, func() {
			body := toResponseBody(test.responseBody)
			length := int64(len(test.responseBody))
			s.mockHttpClient.
				On(Do, mock.MatchedBy(getRequestMatcher(test.accountID)), mock.Anything).
				Return(&http.Response{StatusCode: test.responseStatus, Body: body, ContentLength: length}, nil).
				Once()

			_, actualError := s.accountClient.Fetch(test.accountID)

			s.ErrorIs(test.expectedError, actualError)
		})
	}
}

func (s *accountTestSuite) TestFetchReturnsHttpClientError() {
	accountID := uuid.New()
	expectedError := errors.New("http client error")
	s.mockHttpClient.
		On(Do, mock.MatchedBy(getRequestMatcher(accountID)), mock.Anything).
		Return(nil, expectedError).
		Once()

	_, actualError := s.accountClient.Fetch(accountID)

	s.ErrorIs(expectedError, actualError)
}

func (s *accountTestSuite) TestFetchAccount() {
	accountID := uuid.New()
	expectedAccount := AccountData{
		ID: accountID.String(),
	}
	body, err := json.Marshal(dataContainer{Data: expectedAccount})
	s.Require().NoError(err)

	s.mockHttpClient.
		On(Do, mock.MatchedBy(getRequestMatcher(accountID)), mock.Anything).
		Return(&http.Response{StatusCode: http.StatusOK, Body: toResponseBody(string(body))}, nil).
		Once()

	acc, err := s.accountClient.Fetch(accountID)
	s.NoError(err)
	s.Equal(accountID.String(), acc.ID)
}

func (s *accountTestSuite) TestDeleteVersionedAccountReturnsError_WhenNilUuidGiven() {
	actualError := s.accountClient.DeleteVersion(uuid.Nil, 0)

	s.ErrorIs(ErrNilUUID, actualError)
	s.mockHttpClient.AssertNotCalled(s.T(), Do)
}

func (s *accountTestSuite) TestDeleteVersionedAccountReturnsError() {
	for _, test := range []struct {
		name           string
		accountID      uuid.UUID
		version        uint
		responseStatus int
		responseBody   string
		expectedError  error
	}{
		{
			name:           "account not found",
			accountID:      uuid.New(),
			responseStatus: http.StatusNotFound,
			expectedError:  ErrAccountNotFound,
		}, {
			name:           "invalid account version",
			accountID:      uuid.New(),
			version:        uint(999),
			responseStatus: http.StatusConflict,
			expectedError:  ErrInvalidAccountVersion,
		}, {
			name:           "server error",
			accountID:      uuid.New(),
			responseStatus: http.StatusInternalServerError,
			responseBody:   "{\"error_message\": \"backend error\"}",
			expectedError:  ErrServerError,
		},
		{
			name:           "server bad gateway",
			accountID:      uuid.New(),
			responseStatus: http.StatusBadGateway,
			expectedError:  ErrServerError,
		},
		{
			name:           "server gateway timeout",
			accountID:      uuid.New(),
			responseStatus: http.StatusGatewayTimeout,
			expectedError:  ErrServerError,
		},
		{
			name:           "server unavailable",
			accountID:      uuid.New(),
			responseStatus: http.StatusServiceUnavailable,
			expectedError:  ErrServerUnavailable,
		},
	} {
		s.Run(test.name, func() {
			s.mockHttpClient.
				On(Do, mock.MatchedBy(deleteRequestMatcher(test.accountID, test.version)), mock.Anything).
				Return(&http.Response{StatusCode: test.responseStatus, Body: toResponseBody(test.responseBody)}, nil).
				Once()

			actualError := s.accountClient.DeleteVersion(test.accountID, test.version)

			s.ErrorIs(test.expectedError, actualError)
		})
	}
}

func (s *accountTestSuite) TestDeleteVersionedAccountReturnsHttpClientError() {
	accountID := uuid.New()
	expectedError := errors.New("http client error")
	s.mockHttpClient.
		On(Do, mock.MatchedBy(deleteRequestMatcher(accountID, 0)), mock.Anything).
		Return(nil, expectedError).
		Once()

	actualError := s.accountClient.DeleteVersion(accountID, 0)

	s.ErrorIs(expectedError, actualError)
}

func (s *accountTestSuite) TestDeleteVersionedAccount() {
	accountID := uuid.New()
	s.mockHttpClient.
		On(Do, mock.MatchedBy(deleteRequestMatcher(accountID, 0)), mock.Anything).
		Return(&http.Response{StatusCode: http.StatusNoContent, Body: toResponseBody("")}, nil).
		Once()

	s.NoError(s.accountClient.DeleteVersion(accountID, 0))
}

func (s *accountTestSuite) TestDeleteLatestAccountVersion() {
	accountID := uuid.New()
	version := int64(42)
	expectedAccount := AccountData{
		ID:      accountID.String(),
		Version: &version,
	}
	body, err := json.Marshal(dataContainer{Data: expectedAccount})
	s.Require().NoError(err)

	s.mockHttpClient.
		On(Do, mock.MatchedBy(getRequestMatcher(accountID)), mock.Anything).
		Return(&http.Response{StatusCode: http.StatusOK, Body: toResponseBody(string(body))}, nil).
		Once()

	s.mockHttpClient.
		On(Do, mock.MatchedBy(deleteRequestMatcher(accountID, uint(version))), mock.Anything).
		Return(&http.Response{StatusCode: http.StatusNoContent, Body: toResponseBody("")}, nil).
		Once()

	s.NoError(s.accountClient.Delete(accountID))
	s.mockHttpClient.AssertExpectations(s.T())
}

func postRequestMatcher(data AccountData) func(input *http.Request) bool {
	return func(input *http.Request) bool {
		return input.Method == http.MethodPost &&
			input.URL.String() == testAccountsUrl
	}
}

func getRequestMatcher(expectedAccountID uuid.UUID) func(input *http.Request) bool {
	expectedUrl := fmt.Sprintf("%s/%s", testAccountsUrl, expectedAccountID)
	return func(input *http.Request) bool {
		return input.Method == http.MethodGet &&
			input.URL.String() == expectedUrl
	}
}

func deleteRequestMatcher(expectedAccountID uuid.UUID, expectedVersion uint) func(input *http.Request) bool {
	expectedUrl := fmt.Sprintf("%s/%s?version=%d", testAccountsUrl, expectedAccountID, expectedVersion)
	return func(input *http.Request) bool {
		return input.Method == http.MethodDelete &&
			input.URL.String() == expectedUrl
	}
}

func toStringPtr(b []byte) *string {
	s := string(b)
	return &s
}
