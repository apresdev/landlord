package authx_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/mock"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/stretchr/testify/suite"
)

type UserPasswordAuthTestSuite struct {
	suite.Suite
}

func TestUserPasswordAuthTestSuite(t *testing.T) {
	suite.Run(t, &UserPasswordAuthTestSuite{})
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuth() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPoolClient(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		&cognitoidentityprovider.InitiateAuthOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken:  aws.String("test-access-token"),
				IdToken:      aws.String("test-id-token"),
				RefreshToken: aws.String("test-refresh-token"),
			}}, nil).Once()
	val := mocks.NewTokenValidator(s.T())
	val.EXPECT().ValidateToken(
		ctx, authx.EmptyOpaqueClientID, mock.Anything, "test-access-token").Return(
		&jwt.Token{Raw: "raw-test-access-token"}, nil).Once()
	val.EXPECT().ValidateToken(
		ctx, authx.EmptyOpaqueClientID, mock.Anything, "test-id-token").Return(
		&jwt.Token{Raw: "raw-test-id-token"}, nil).Once()
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	valURL, err := url.Parse("https://example.com/validation-url")
	s.Assert().NoError(err)
	tokens, err := authx.UserPasswordAuth(
		ctx, cogIdp, val, valURL, &up, "example@example.com", "sekrit")
	s.Assert().NoError(err)
	s.Assert().Equal("raw-test-access-token", tokens.AccessToken.Raw)
	s.Assert().Equal("raw-test-id-token", tokens.IdToken.Raw)
	s.Assert().Equal("test-refresh-token", tokens.RefreshToken)
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuthError() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	val := mocks.NewTokenValidator(s.T())
	valURL, err := url.Parse("https://example.com/validation-url")
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.UserPasswordAuth(
		ctx, cogIdp, val, valURL, &up, "example@example.com", "sekrit")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
	s.Assert().Equal("test error", err.Error())
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuthError2() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPoolClient(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		nil, errors.New("test error")).Once()
	val := mocks.NewTokenValidator(s.T())
	valURL, err := url.Parse("https://example.com/validation-url")
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.UserPasswordAuth(
		ctx, cogIdp, val, valURL, &up, "example@example.com", "sekrit")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
	s.Assert().Equal("test error", err.Error())
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuthError3() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPoolClient(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		&cognitoidentityprovider.InitiateAuthOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken:  aws.String("test-access-token"),
				IdToken:      aws.String("test-id-token"),
				RefreshToken: aws.String("test-refresh-token"),
			}}, nil).Once()
	val := mocks.NewTokenValidator(s.T())
	val.EXPECT().ValidateToken(
		ctx, authx.EmptyOpaqueClientID, mock.Anything, "test-access-token").Return(
		nil, errors.New("test error")).Once()
	valURL, err := url.Parse("https://example.com/validation-url")
	s.Assert().NoError(err)
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.UserPasswordAuth(
		ctx, cogIdp, val, valURL, &up, "example@example.com", "sekrit")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
	s.Assert().Equal("test error", err.Error())
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuthError4() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPoolClient(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolClientOutput{
			UserPoolClient: &types.UserPoolClientType{
				ClientId:     aws.String("test-client-id1"),
				ClientSecret: aws.String("test-client-secret")}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		&cognitoidentityprovider.InitiateAuthOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken:  aws.String("test-access-token"),
				IdToken:      aws.String("test-id-token"),
				RefreshToken: aws.String("test-refresh-token"),
			}}, nil).Once()
	val := mocks.NewTokenValidator(s.T())
	val.EXPECT().ValidateToken(
		ctx, authx.EmptyOpaqueClientID, mock.Anything, "test-access-token").Return(
		&jwt.Token{}, nil).Once()
	val.EXPECT().ValidateToken(
		ctx, authx.EmptyOpaqueClientID, mock.Anything, "test-id-token").Return(
		nil, errors.New("test error")).Once()
	valURL, err := url.Parse("https://example.com/validation-url")
	s.Assert().NoError(err)
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.UserPasswordAuth(
		ctx, cogIdp, val, valURL, &up, "example@example.com", "sekrit")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
	s.Assert().Equal("test error", err.Error())
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuthNoArgs() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	val := mocks.NewTokenValidator(s.T())
	ctx := context.Background()
	tokens, err := authx.UserPasswordAuth(
		ctx, cogIdp, val, nil, nil, "", "")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuth() {
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
		ctx, authx.EmptyOpaqueClientID, mock.Anything, "test-access-token").Return(
		&jwt.Token{}, nil).Once()
	val.EXPECT().ValidateToken(
		ctx, authx.EmptyOpaqueClientID, mock.Anything, "test-id-token").Return(
		&jwt.Token{}, nil).Once()
	valURL, err := url.Parse("https://example.com/validation-url")
	s.Assert().NoError(err)
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.RefreshAuth(
		ctx, cogIdp, val, valURL, &up, "test-refresh-token")
	s.Assert().NoError(err)
	s.Assert().Equal("test-access-token", tokens.AccessToken)
	s.Assert().Equal("test-id-token", tokens.IdToken)
	s.Assert().Equal("test-refresh-token", tokens.RefreshToken)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuthError2() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	val := mocks.NewTokenValidator(s.T())
	valURL, err := url.Parse("https://example.com/validation-url")
	s.Assert().NoError(err)
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.RefreshAuth(
		ctx, cogIdp, val, valURL, &up, "test-refresh-token")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuthError3() {
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
	tokens, err := authx.RefreshAuth(
		ctx, cogIdp, val, valURL, &up, "test-refresh-token")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuthError4() {
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
		ctx, authx.EmptyOpaqueClientID, valURL, "test-access-token").Return(
		nil, errors.New("test error")).Once()
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.RefreshAuth(
		ctx, cogIdp, val, valURL, &up, "test-refresh-token")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuthError5() {
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
		ctx, authx.EmptyOpaqueClientID, valURL, "test-access-token").Return(
		&jwt.Token{}, nil).Once()
	val.EXPECT().ValidateToken(
		ctx, authx.EmptyOpaqueClientID, valURL, "test-id-token").Return(
		nil, errors.New("test error")).Once()
	up := types.UserPoolType{Id: aws.String("test-user-pool1")}
	tokens, err := authx.RefreshAuth(
		ctx, cogIdp, val, valURL, &up, "test-refresh-token")
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}
