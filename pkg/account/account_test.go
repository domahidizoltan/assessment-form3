package account

import (
	"errors"
	"fmt"
	"form3interview/internal/mocks"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	Do = "Do"
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

func (s *accountTestSuite) TestDeleteAccountReturnsError_WhenNilUuidGiven() {
	actualError := s.accountClient.Delete(uuid.Nil, 0)

	s.ErrorIs(ErrNilUuid, actualError)
	s.mockHttpClient.AssertNotCalled(s.T(), Do)
}

func (s *accountTestSuite) TestDeleteAccountReturnsError() {
	missingAccountID := uuid.Must(uuid.NewUUID())
	invalidVersion := uint(999)
	for _, test := range []struct {
		name          string
		accountID     uuid.UUID
		version       uint
		returnStatus  int
		expectedError error
	}{
		{name: "account not found", accountID: missingAccountID, version: 0, returnStatus: http.StatusNotFound, expectedError: ErrAccountNotFound},
		{name: "invalid account version", accountID: uuid.Must(uuid.NewUUID()), version: invalidVersion, returnStatus: http.StatusConflict, expectedError: ErrInvalidAccountVersion},
	} {
		s.Run(test.name, func() {
			s.mockHttpClient.
				On(Do, mock.MatchedBy(deleteRequestMatcher(test.accountID, test.version))).
				Return(&http.Response{StatusCode: test.returnStatus}, nil).
				Once()

			actualError := s.accountClient.Delete(test.accountID, test.version)

			s.ErrorIs(test.expectedError, actualError)
		})
	}
}

func (s *accountTestSuite) TestDeleteAccountReturnsHttpError() {
	expectedError := errors.New("http error")
	s.mockHttpClient.
		On(Do, mock.Anything, mock.Anything).
		Return(nil, expectedError).
		Once()

	actualError := s.accountClient.Delete(uuid.Must(uuid.NewUUID()), 0)

	s.ErrorIs(expectedError, actualError)
}

func (s *accountTestSuite) TestDeleteAccount() {
	accountID := uuid.Must(uuid.NewUUID())
	s.mockHttpClient.
		On(Do, mock.MatchedBy(deleteRequestMatcher(accountID, 0))).
		Return(&http.Response{StatusCode: http.StatusNoContent}, nil).
		Once()

	s.NoError(s.accountClient.Delete(accountID, 0))
}

func deleteRequestMatcher(expectedAccountID uuid.UUID, expectedVersion uint) func(input *http.Request) bool {
	expectedUrl := fmt.Sprintf("%s/%s?version=%d", accountsUrl, expectedAccountID, expectedVersion)
	return func(input *http.Request) bool {
		return input.Method == http.MethodDelete &&
			input.URL.String() == expectedUrl
	}
}
