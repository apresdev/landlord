package authx_test

import (
	"testing"

	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/apptest"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/stretchr/testify/suite"
)

// UserPoolClientIntegrationTestSuite runs against a local AWS stand-in
// the AWS Emulator and skips when it is not running.
type UserPoolClientIntegrationTestSuite struct {
	suite.Suite
	s appenv.Settings
}

func TestUserPoolClientIntegrationTestSuite(t *testing.T) {
	suite.Run(t, &UserPoolClientIntegrationTestSuite{s: appenv.MustTestSettings()})
}

func (s *UserPoolClientIntegrationTestSuite) SetupSuite() {
	if !apptest.InitEmulator(s.s) {
		s.T().Skip("AWS emulator not running")
	}
}

func (s *UserPoolClientIntegrationTestSuite) SetupTest() {
	apptest.ResetEmulator(s.s)
}

// TestSecondClientInPool reproduces a pool that gains a second app client
// (e.g. for an MCP server) after Landlord is set up.
func (s *UserPoolClientIntegrationTestSuite) TestSecondClientInPool() {
	a := appcore.MustWithSettings(s.s)
	up, err := authx.GetUserPoolByName(a.Ctx(), a.CogIdp(), s.s.UserPoolName())
	s.Require().NoError(err)
	poolID := aws.ToString(up.Id)

	landlordClient, err := authx.GetUserPoolClient(a.Ctx(), a.CogIdp(), poolID, "")
	s.Require().NoError(err)

	other, err := cognitoidentityprovider.NewFromConfig(a.Cfg()).CreateUserPoolClient(a.Ctx(),
		&cognitoidentityprovider.CreateUserPoolClientInput{
			ClientName: aws.String("Prod-ClearSight-MCP"),
			UserPoolId: up.Id,
		})
	s.Require().NoError(err)

	_, err = authx.GetUserPoolClient(a.Ctx(), a.CogIdp(), poolID, "")
	s.Assert().ErrorContains(err, "more than one user pool client")

	byName, err := authx.GetUserPoolClient(
		a.Ctx(), a.CogIdp(), poolID, aws.ToString(landlordClient.ClientName))
	s.Require().NoError(err)
	s.Assert().Equal(aws.ToString(landlordClient.ClientId), aws.ToString(byName.ClientId))

	byID, err := authx.GetUserPoolClient(
		a.Ctx(), a.CogIdp(), poolID, aws.ToString(landlordClient.ClientId))
	s.Require().NoError(err)
	s.Assert().NotEqual(aws.ToString(other.UserPoolClient.ClientId), aws.ToString(byID.ClientId))
	s.Assert().Equal(aws.ToString(landlordClient.ClientId), aws.ToString(byID.ClientId))
}
