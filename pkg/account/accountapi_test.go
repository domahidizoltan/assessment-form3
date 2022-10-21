package account

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type accountTestSuite struct {
	suite.Suite
}

func TestAccountTestSuite(t *testing.T) {
	suite.Run(t, new(accountTestSuite))
}

func (s accountTestSuite) TestFetchAccountData() {
	s.T().Setenv("FORM3_BASE_URL", "http://localhost:8080/v1")
	c, _ := NewClient()
	d, err := c.Fetch(uuid.MustParse("ad27e265-9605-4b4b-a0e5-3003ea9cc4dc"))
	fmt.Printf("response %+v err %+v", d, err)
}
