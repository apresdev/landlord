package authx_test

import (
	"context"
	"testing"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type UserPoolClientTestSuite struct {
	suite.Suite
}

func TestUserPoolClientTestSuite(t *testing.T) {
	suite.Run(t, &UserPoolClientTestSuite{})
}

func (s *UserPoolClientTestSuite) mockClients(
	clients ...types.UserPoolClientDescription) (*mocks.AWSCognitoIdentityProvider, context.Context) {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{UserPoolClients: clients}, nil)
	return cogIdp, ctx
}

func (s *UserPoolClientTestSuite) expectDescribe(
	cogIdp *mocks.AWSCognitoIdentityProvider, ctx context.Context, id string) {
	cogIdp.EXPECT().DescribeUserPoolClient(ctx, mock.Anything).RunAndReturn(
		func(_ context.Context, in *cognitoidentityprovider.DescribeUserPoolClientInput,
			_ ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.DescribeUserPoolClientOutput, error) {
			s.Assert().Equal(id, aws.ToString(in.ClientId))
			return &cognitoidentityprovider.DescribeUserPoolClientOutput{
				UserPoolClient: &types.UserPoolClientType{ClientId: in.ClientId},
			}, nil
		})
}

var twoClients = []types.UserPoolClientDescription{
	{ClientId: aws.String("mcp-id"), ClientName: aws.String("Prod-ClearSight-MCP")},
	{ClientId: aws.String("ll-id"), ClientName: aws.String("Prod-landlord-default-user-pool-client")},
}

func (s *UserPoolClientTestSuite) TestSingleClientNoPreference() {
	cogIdp, ctx := s.mockClients(twoClients[1])
	s.expectDescribe(cogIdp, ctx, "ll-id")
	c, err := authx.GetUserPoolClient(ctx, cogIdp, "pool", "")
	s.Require().NoError(err)
	s.Assert().Equal("ll-id", aws.ToString(c.ClientId))
}

func (s *UserPoolClientTestSuite) TestSingleClientPreferenceMismatchFallsBack() {
	cogIdp, ctx := s.mockClients(twoClients[1])
	s.expectDescribe(cogIdp, ctx, "ll-id")
	c, err := authx.GetUserPoolClient(ctx, cogIdp, "pool", "some-other-name")
	s.Require().NoError(err)
	s.Assert().Equal("ll-id", aws.ToString(c.ClientId))
}

func (s *UserPoolClientTestSuite) TestMultipleClientsPreferredByID() {
	cogIdp, ctx := s.mockClients(twoClients...)
	s.expectDescribe(cogIdp, ctx, "ll-id")
	c, err := authx.GetUserPoolClient(ctx, cogIdp, "pool", "ll-id")
	s.Require().NoError(err)
	s.Assert().Equal("ll-id", aws.ToString(c.ClientId))
}

func (s *UserPoolClientTestSuite) TestMultipleClientsPreferredByName() {
	cogIdp, ctx := s.mockClients(twoClients...)
	s.expectDescribe(cogIdp, ctx, "ll-id")
	c, err := authx.GetUserPoolClient(
		ctx, cogIdp, "pool", "Prod-landlord-default-user-pool-client")
	s.Require().NoError(err)
	s.Assert().Equal("ll-id", aws.ToString(c.ClientId))
}

func (s *UserPoolClientTestSuite) TestMultipleClientsNoPreferenceErrors() {
	cogIdp, ctx := s.mockClients(twoClients...)
	c, err := authx.GetUserPoolClient(ctx, cogIdp, "pool", "")
	s.Assert().ErrorContains(err, "more than one user pool client")
	s.Assert().Nil(c)
}

func (s *UserPoolClientTestSuite) TestMultipleClientsNoMatchErrors() {
	cogIdp, ctx := s.mockClients(twoClients...)
	c, err := authx.GetUserPoolClient(ctx, cogIdp, "pool", "nope")
	s.Assert().ErrorContains(err, "none matches")
	s.Assert().Nil(c)
}

func (s *UserPoolClientTestSuite) TestNoClientsErrors() {
	cogIdp, ctx := s.mockClients()
	_, err := authx.GetUserPoolClient(ctx, cogIdp, "pool", "x")
	s.Assert().ErrorContains(err, "no user pool clients")
}
