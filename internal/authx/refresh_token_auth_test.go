package authx_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/stretchr/testify/suite"
)

type RefreshTokenAuthTestSuite struct {
	suite.Suite
}

func TestRefreshTokenAuthTestSuite(t *testing.T) {
	suite.Run(t, &RefreshTokenAuthTestSuite{})
}

func (s *RefreshTokenAuthTestSuite) TestRefreshTokenAuth() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		&cognitoidentityprovider.InitiateAuthOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken: aws.String("test-access-token"),
				IdToken:     aws.String("test-id-token"),
			}}, nil).Once()
	val := mocks.NewTokenValidator(s.T())
	val.EXPECT().ValidateToken(
		ctx, jwx.EmptyOpaqueClientID, mock.Anything, "test-access-token").Return(
		&jwt.Token{}, nil).Once()
	val.EXPECT().ValidateToken(
		ctx, jwx.EmptyOpaqueClientID, mock.Anything, "test-id-token").Return(
		&jwt.Token{}, nil).Once()
	valURL, err := url.Parse("https://example.com/validation-url")
	s.Assert().NoError(err)
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.RefreshTokenAuth(
		ctx, cogIdp, val, valURL, &up, "test-refresh-token")
	s.Assert().NoError(err)
	s.Assert().Equal("test-access-token", tokens.AccessToken)
	s.Assert().Equal("test-id-token", tokens.IdToken)
	s.Assert().Equal("test-refresh-token", tokens.RefreshToken)
}

func (s *RefreshTokenAuthTestSuite) TestRefreshTokenAuthError2() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	val := mocks.NewTokenValidator(s.T())
	valURL, err := url.Parse("https://example.com/validation-url")
	s.Assert().NoError(err)
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.RefreshTokenAuth(
		ctx, cogIdp, val, valURL, &up, "test-refresh-token")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *RefreshTokenAuthTestSuite) TestRefreshTokenAuthError3() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	val := mocks.NewTokenValidator(s.T())
	valURL, err := url.Parse("https://example.com/validation-url")
	s.Assert().NoError(err)
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.RefreshTokenAuth(
		ctx, cogIdp, val, valURL, &up, "test-refresh-token")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *RefreshTokenAuthTestSuite) TestRefreshTokenAuthError4() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		&cognitoidentityprovider.InitiateAuthOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken: aws.String("test-access-token"),
				IdToken:     aws.String("test-id-token"),
			}}, nil).Once()
	valURL, err := url.Parse("https://example.com/validation-url")
	s.Assert().NoError(err)
	val := mocks.NewTokenValidator(s.T())
	val.EXPECT().ValidateToken(
		ctx, jwx.EmptyOpaqueClientID, valURL, "test-access-token").Return(
		nil, errors.New("test error")).Once()
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.RefreshTokenAuth(
		ctx, cogIdp, val, valURL, &up, "test-refresh-token")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *RefreshTokenAuthTestSuite) TestRefreshTokenAuthError5() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		&cognitoidentityprovider.InitiateAuthOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken: aws.String("test-access-token"),
				IdToken:     aws.String("test-id-token"),
			}}, nil).Once()
	val := mocks.NewTokenValidator(s.T())
	valURL, err := url.Parse("https://example.com/validation-url")
	s.Assert().NoError(err)
	val.EXPECT().ValidateToken(
		ctx, jwx.EmptyOpaqueClientID, valURL, "test-access-token").Return(
		&jwt.Token{}, nil).Once()
	val.EXPECT().ValidateToken(
		ctx, jwx.EmptyOpaqueClientID, valURL, "test-id-token").Return(
		nil, errors.New("test error")).Once()
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.RefreshTokenAuth(
		ctx, cogIdp, val, valURL, &up, "test-refresh-token")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}
