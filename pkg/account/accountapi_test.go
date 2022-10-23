package account

import (
	"encoding/json"
	"form3interview/pkg/config"
	"form3interview/pkg/requestenricher"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	intTestOrganisationID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	intTestAccountID      = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	intTestAttributes     *AccountAttributes
)

type accountApiTestSuite struct {
	suite.Suite
	db                   *gorm.DB
	originalGenerateFunc func() (uuid.UUID, error)
	accountClient        *accountClient
}

func TestAccountApiTestSuite(t *testing.T) {
	suite.Run(t, new(accountApiTestSuite))
}

func (s *accountApiTestSuite) SetupSuite() {
	var err error
	s.db, err = gorm.Open(postgres.Open("host=localhost user=root password=password dbname=interview_accountapi"))
	s.Require().NoError(err)

	fixture, err := os.ReadFile("testdata/account_attributes.json")
	s.Require().NoError(err)

	s.Require().NoError(json.Unmarshal(fixture, &intTestAttributes))

	s.originalGenerateFunc = generateUUID
	generateUUID = func() (uuid.UUID, error) { return intTestAccountID, nil }

	s.accountClient, err = NewClient(
		config.WithBaseUrl("http://localhost:8080/v1"),
		config.WithOrganisationID(intTestOrganisationID),
	)
	s.Require().NoError(err)

}

func (s *accountApiTestSuite) SetupTest() {
	s.Require().NoError(s.db.Exec("DELETE FROM \"Account\" WHERE id=?", intTestAccountID).Error)
}

func (s *accountApiTestSuite) TearDownSuite() {
	generateUUID = s.originalGenerateFunc
}

func (s accountApiTestSuite) Test1_CreateAccount() {
	actualData, err := s.accountClient.Create(*intTestAttributes)
	s.NoError(err)
	s.assertAccountData(actualData)
}

func (s accountApiTestSuite) Test2_FetchAccount() {
	_, err := s.accountClient.Create(*intTestAttributes)
	s.NoError(err)

	actualData, err := s.accountClient.Fetch(intTestAccountID)
	s.NoError(err)
	s.assertAccountData(actualData)
}

func (s accountApiTestSuite) Test3_DeleteAccount() {
	_, err := s.accountClient.Create(*intTestAttributes)
	s.NoError(err)

	s.NoError(s.accountClient.Delete(intTestAccountID))
	_, err = s.accountClient.Fetch(intTestAccountID)
	s.ErrorIs(err, ErrAccountNotFound)
}

func (s accountApiTestSuite) Test4_EnrichedRequest() {
	var start time.Time
	beforeHookCalled := false
	afterHookCalled := false
	en := requestenricher.RequestEnricher{
		BeforeHook: func() {
			beforeHookCalled = true
			start = time.Now()
		},
		AfterHook: func(r *http.Response) {
			afterHookCalled = true
			log.Info().Msgf("Request took %s and returned with status %d", time.Since(start), r.StatusCode)
		},
	}

	actualData, err := s.accountClient.Create(*intTestAttributes, en)
	s.NoError(err)
	s.True(beforeHookCalled)
	s.True(afterHookCalled)
	s.assertAccountData(actualData)
}

func (s accountApiTestSuite) assertAccountData(data *AccountData) {
	s.NotNil(data)

	s.Equal(intTestAccountID.String(), data.ID)
	s.Equal(intTestOrganisationID.String(), data.OrganisationID)
	s.Equal(accountsType, data.Type)
	s.Equal(int64(0), *data.Version)

	atr := data.Attributes
	s.Equal("Personal", *atr.AccountClassification)
	s.Equal(true, *atr.AccountMatchingOptOut)
	s.Equal("0500013M026", atr.AccountNumber)
	s.Len(atr.AlternativeNames, 1)
	s.Equal("testAltName", atr.AlternativeNames[0])
	s.Equal("20041", atr.BankID)
	s.Equal("FR", atr.BankIDCode)
	s.Equal("EUR", atr.BaseCurrency)
	s.Equal("NWBKFR42", atr.Bic)
	s.Equal("FR", *atr.Country)
	s.Equal("FR1420041010050500013M02606", atr.Iban)
	s.Equal(true, *atr.JointAccount)
	s.Len(atr.Name, 1)
	s.Equal("testName", atr.Name[0])
	s.Equal("secID", atr.SecondaryIdentification)
	s.Equal("confirmed", *atr.Status)
	s.Equal(true, *atr.Switched)
}
