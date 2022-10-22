package account

import (
	"encoding/json"
	"errors"
	"fmt"
	"form3interview/internal/mocks"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	Do  = "Do"
	Get = "Get"
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
	s.accountClient = accountClient{
		client: s.mockHttpClient,
	}
}

func (s *accountTestSuite) TestFetchAccountAccountReturnsError_WhenNilUuidGiven() {
	_, actualError := s.accountClient.Fetch(uuid.Nil)

	s.ErrorIs(ErrNilUuid, actualError)
	s.mockHttpClient.AssertNotCalled(s.T(), Get)
}

func (s *accountTestSuite) TestFetchAccountReturnsError() {
	for _, test := range []struct {
		name           string
		accountID      uuid.UUID
		responseStatus int
		responseBody   string
		expectedError  error
	}{
		{
			name:           "account not found",
			accountID:      newUuid(),
			responseStatus: http.StatusNotFound,
			expectedError:  ErrAccountNotFound,
		}, {
			name:           "server error",
			accountID:      newUuid(),
			responseStatus: http.StatusInternalServerError,
			responseBody:   "{\"error_message\": \"backend error\"}",
			expectedError:  ErrServerError,
		},
		{
			name:           "unexpected server response",
			accountID:      newUuid(),
			responseStatus: http.StatusTeapot,
			responseBody:   "oops",
			expectedError:  ErrUnexpectedServerResponse,
		},
	} {
		s.Run(test.name, func() {
			body := toResponseBody(test.responseBody)
			length := int64(len(test.responseBody))
			s.mockHttpClient.
				On(Get, accountsUrl+test.accountID.String()).
				Return(&http.Response{StatusCode: test.responseStatus, Body: body, ContentLength: length}, nil).
				Once()

			_, actualError := s.accountClient.Fetch(test.accountID)

			s.ErrorIs(test.expectedError, actualError)
		})
	}
}

func (s *accountTestSuite) TestFetchAccountReturnsHttpError() {
	expectedError := errors.New("http error")
	s.mockHttpClient.
		On(Get, mock.Anything).
		Return(nil, expectedError).
		Once()

	_, actualError := s.accountClient.Fetch(newUuid())

	s.ErrorIs(expectedError, actualError)
}

func (s *accountTestSuite) TestFetchAccount() {
	accountID := newUuid()
	expectedAccount := AccountData{
		ID: accountID.String(),
	}
	body, err := json.Marshal(responseContainer{Data: expectedAccount})
	s.Require().NoError(err)

	s.mockHttpClient.
		On(Get, accountsUrl+accountID.String()).
		Return(&http.Response{StatusCode: http.StatusOK, Body: toResponseBody(string(body))}, nil).
		Once()

	acc, err := s.accountClient.Fetch(accountID)
	s.NoError(err)
	s.Equal(acc.ID, accountID.String())
}

func (s *accountTestSuite) TestDeleteVersionedAccountReturnsError_WhenNilUuidGiven() {
	actualError := s.accountClient.DeleteVersion(uuid.Nil, 0)

	s.ErrorIs(ErrNilUuid, actualError)
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
			accountID:      newUuid(),
			responseStatus: http.StatusNotFound,
			expectedError:  ErrAccountNotFound,
		}, {
			name:           "invalid account version",
			accountID:      newUuid(),
			version:        uint(999),
			responseStatus: http.StatusConflict,
			expectedError:  ErrInvalidAccountVersion,
		}, {
			name:           "server error",
			accountID:      newUuid(),
			responseStatus: http.StatusInternalServerError,
			responseBody:   "{\"error_message\": \"backend error\"}",
			expectedError:  ErrServerError,
		},
	} {
		s.Run(test.name, func() {
			s.mockHttpClient.
				On(Do, mock.MatchedBy(deleteRequestMatcher(test.accountID, test.version))).
				Return(&http.Response{StatusCode: test.responseStatus, Body: toResponseBody(test.responseBody)}, nil).
				Once()

			actualError := s.accountClient.DeleteVersion(test.accountID, test.version)

			s.ErrorIs(test.expectedError, actualError)
		})
	}
}

func (s *accountTestSuite) TestDeleteVersionedAccountReturnsHttpError() {
	expectedError := errors.New("http error")
	s.mockHttpClient.
		On(Do, mock.Anything, mock.Anything).
		Return(nil, expectedError).
		Once()

	actualError := s.accountClient.DeleteVersion(newUuid(), 0)

	s.ErrorIs(expectedError, actualError)
}

func (s *accountTestSuite) TestDeleteVersionedAccount() {
	accountID := newUuid()
	s.mockHttpClient.
		On(Do, mock.MatchedBy(deleteRequestMatcher(accountID, 0))).
		Return(&http.Response{StatusCode: http.StatusNoContent, Body: toResponseBody("")}, nil).
		Once()

	s.NoError(s.accountClient.DeleteVersion(accountID, 0))
}

func (s *accountTestSuite) TestDeleteLatestAccountVersion() {
	accountID := newUuid()
	version := int64(42)
	expectedAccount := AccountData{
		ID:      accountID.String(),
		Version: &version,
	}
	body, err := json.Marshal(responseContainer{Data: expectedAccount})
	s.Require().NoError(err)

	s.mockHttpClient.
		On(Get, accountsUrl+accountID.String()).
		Return(&http.Response{StatusCode: http.StatusOK, Body: toResponseBody(string(body))}, nil).
		Once()

	s.mockHttpClient.
		On(Do, mock.MatchedBy(deleteRequestMatcher(accountID, uint(version)))).
		Return(&http.Response{StatusCode: http.StatusNoContent, Body: toResponseBody("")}, nil).
		Once()

	s.NoError(s.accountClient.Delete(accountID))
	s.mockHttpClient.AssertExpectations(s.T())
}

func deleteRequestMatcher(expectedAccountID uuid.UUID, expectedVersion uint) func(input *http.Request) bool {
	expectedUrl := fmt.Sprintf("%s/%s?version=%d", accountsUrl, expectedAccountID, expectedVersion)
	return func(input *http.Request) bool {
		return input.Method == http.MethodDelete &&
			input.URL.String() == expectedUrl
	}
}

func newUuid() uuid.UUID {
	return uuid.Must(uuid.NewUUID())
}

func toResponseBody(body string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(body))
}
