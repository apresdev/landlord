package authx_test

import (
	"context"
	"errors"
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
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool1"),
			}}, nil).Once()
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		&cognitoidentityprovider.InitiateAuthOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken:  aws.String("test-access-token"),
				IdToken:      aws.String("test-id-token"),
				RefreshToken: aws.String("test-refresh-token"),
			}}, nil).Once()
	val := mocks.NewTokenValidator(s.T())
	val.EXPECT().ValidateToken(
		ctx, "us-east-1", mock.Anything, "test-access-token").Return(
		&jwt.Token{}, nil).Once()
	val.EXPECT().ValidateToken(
		ctx, "us-east-1", mock.Anything, "test-id-token").Return(
		&jwt.Token{}, nil).Once()
	tokens, err := authx.UserPasswordAuth(ctx, cogIdp, val, authx.UserPasswordAuthArgs{
		Region:   "us-east-1",
		UserPool: "test-user-pool1",
		Email:    "example@example.com",
		Password: "sekrit",
	})
	s.Assert().NoError(err)
	s.Assert().Equal("test-access-token", tokens.AccessToken)
	s.Assert().Equal("test-id-token", tokens.IdToken)
	s.Assert().Equal("test-refresh-token", tokens.RefreshToken)
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuthError() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool1"),
			}}, nil).Once()
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	val := mocks.NewTokenValidator(s.T())
	tokens, err := authx.UserPasswordAuth(ctx, cogIdp, val, authx.UserPasswordAuthArgs{
		Region:   "us-east-1",
		UserPool: "test-user-pool1",
		Email:    "example@example.com",
		Password: "sekrit",
	})
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
	s.Assert().Equal("test error", err.Error())
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuthError2() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool1"),
			}}, nil).Once()
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		nil, errors.New("test error")).Once()
	val := mocks.NewTokenValidator(s.T())
	tokens, err := authx.UserPasswordAuth(ctx, cogIdp, val, authx.UserPasswordAuthArgs{
		Region:   "us-east-1",
		UserPool: "test-user-pool1",
		Email:    "example@example.com",
		Password: "sekrit",
	})
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
	s.Assert().Equal("test error", err.Error())
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuthError3() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool1"),
			}}, nil).Once()
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		&cognitoidentityprovider.InitiateAuthOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken:  aws.String("test-access-token"),
				IdToken:      aws.String("test-id-token"),
				RefreshToken: aws.String("test-refresh-token"),
			}}, nil).Once()
	val := mocks.NewTokenValidator(s.T())
	val.EXPECT().ValidateToken(
		ctx, "us-east-1", mock.Anything, "test-access-token").Return(
		nil, errors.New("test error")).Once()
	tokens, err := authx.UserPasswordAuth(ctx, cogIdp, val, authx.UserPasswordAuthArgs{
		Region:   "us-east-1",
		UserPool: "test-user-pool1",
		Email:    "example@example.com",
		Password: "sekrit",
	})
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
	s.Assert().Equal("test error", err.Error())
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuthError4() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool1"),
			}}, nil).Once()
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		&cognitoidentityprovider.InitiateAuthOutput{
			AuthenticationResult: &types.AuthenticationResultType{
				AccessToken:  aws.String("test-access-token"),
				IdToken:      aws.String("test-id-token"),
				RefreshToken: aws.String("test-refresh-token"),
			}}, nil).Once()
	val := mocks.NewTokenValidator(s.T())
	val.EXPECT().ValidateToken(
		ctx, "us-east-1", mock.Anything, "test-access-token").Return(
		&jwt.Token{}, nil).Once()
	val.EXPECT().ValidateToken(
		ctx, "us-east-1", mock.Anything, "test-id-token").Return(
		nil, errors.New("test error")).Once()
	tokens, err := authx.UserPasswordAuth(ctx, cogIdp, val, authx.UserPasswordAuthArgs{
		Region:   "us-east-1",
		UserPool: "test-user-pool1",
		Email:    "example@example.com",
		Password: "sekrit",
	})
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
	s.Assert().Equal("test error", err.Error())
}

func (s *UserPasswordAuthTestSuite) TestUserPasswordAuthNoArgs() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	val := mocks.NewTokenValidator(s.T())
	ctx := context.Background()
	tokens, err := authx.UserPasswordAuth(ctx, cogIdp, val, authx.UserPasswordAuthArgs{})
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuth() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool1"),
			}}, nil).Once()
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
		ctx, "us-east-1", mock.Anything, "test-access-token").Return(
		&jwt.Token{}, nil).Once()
	val.EXPECT().ValidateToken(
		ctx, "us-east-1", mock.Anything, "test-id-token").Return(
		&jwt.Token{}, nil).Once()
	tokens, err := authx.RefreshAuth(ctx, cogIdp, val, authx.RefreshAuthArgs{
		Region:       "us-east-1",
		UserPool:     "test-user-pool1",
		RefreshToken: "test-refresh-token",
	})
	s.Assert().NoError(err)
	s.Assert().Equal("test-access-token", tokens.AccessToken)
	s.Assert().Equal("test-id-token", tokens.IdToken)
	s.Assert().Equal("test-refresh-token", tokens.RefreshToken)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuthError() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	val := mocks.NewTokenValidator(s.T())
	tokens, err := authx.RefreshAuth(ctx, cogIdp, val, authx.RefreshAuthArgs{
		Region:       "us-east-1",
		UserPool:     "test-user-pool1",
		RefreshToken: "test-refresh-token",
	})
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuthError2() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool1"),
			}}, nil).Once()
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	val := mocks.NewTokenValidator(s.T())
	tokens, err := authx.RefreshAuth(ctx, cogIdp, val, authx.RefreshAuthArgs{
		Region:       "us-east-1",
		UserPool:     "test-user-pool1",
		RefreshToken: "test-refresh-token",
	})
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuthError3() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool1"),
			}}, nil).Once()
	cogIdp.EXPECT().ListUserPoolClients(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolClientsOutput{
			UserPoolClients: []types.UserPoolClientDescription{
				{ClientId: aws.String("test-client-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().InitiateAuth(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	val := mocks.NewTokenValidator(s.T())
	tokens, err := authx.RefreshAuth(ctx, cogIdp, val, authx.RefreshAuthArgs{
		Region:       "us-east-1",
		UserPool:     "test-user-pool1",
		RefreshToken: "test-refresh-token",
	})
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuthError4() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool1"),
			}}, nil).Once()
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
		ctx, "us-east-1", mock.Anything, "test-access-token").Return(
		nil, errors.New("test error")).Once()
	tokens, err := authx.RefreshAuth(ctx, cogIdp, val, authx.RefreshAuthArgs{
		Region:       "us-east-1",
		UserPool:     "test-user-pool1",
		RefreshToken: "test-refresh-token",
	})
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}

func (s *UserPasswordAuthTestSuite) TestRefreshAuthError5() {
	ctx := context.Background()
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Id: aws.String("test-user-pool1"),
			}}, nil).Once()
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
		ctx, "us-east-1", mock.Anything, "test-access-token").Return(
		&jwt.Token{}, nil).Once()
	val.EXPECT().ValidateToken(
		ctx, "us-east-1", mock.Anything, "test-id-token").Return(
		nil, errors.New("test error")).Once()
	tokens, err := authx.RefreshAuth(ctx, cogIdp, val, authx.RefreshAuthArgs{
		Region:       "us-east-1",
		UserPool:     "test-user-pool1",
		RefreshToken: "test-refresh-token",
	})
	s.Assert().Error(err)
	s.Assert().Nil(tokens)
}
