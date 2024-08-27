package authx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apresdev/landlord/internal/appenv"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/mocks"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/suite"
)

type UserPoolTestSuite struct {
	suite.Suite
}

func TestUserPoolTestSuite(t *testing.T) {
	suite.Run(t, &UserPoolTestSuite{})
}

func (s *UserPoolTestSuite) TestGetUserPoolDescriptionsEmpty() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{}, nil)
	v, err := authx.GetUserPoolDescriptions(ctx, cogIdp)
	s.Assert().NoError(err)
	s.Assert().Empty(v)
}

func (s *UserPoolTestSuite) TestGetUserPoolDescriptions() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Id: aws.String("1")},
				{Id: aws.String("2")},
			}}, nil)
	v, err := authx.GetUserPoolDescriptions(ctx, cogIdp)
	s.Assert().NoError(err)
	s.Assert().NotEmpty(v)
	s.Assert().Len(v, 2)
	s.Assert().Equal("1", *v[0].Id)
	s.Assert().Equal("2", *v[1].Id)
}

func (s *UserPoolTestSuite) TestGetUserPoolDescriptionsPaged() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			NextToken: aws.String("n"),
			UserPools: []types.UserPoolDescriptionType{
				{Id: aws.String("1")},
				{Id: aws.String("2")},
			}}, nil).Once()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Id: aws.String("3")},
				{Id: aws.String("4")},
			}}, nil)
	v, err := authx.GetUserPoolDescriptions(ctx, cogIdp)
	s.Assert().NoError(err)
	s.Assert().NotEmpty(v)
	s.Assert().Len(v, 4)
	s.Assert().Equal("1", *v[0].Id)
	s.Assert().Equal("2", *v[1].Id)
	s.Assert().Equal("3", *v[2].Id)
	s.Assert().Equal("4", *v[3].Id)
}

func (s *UserPoolTestSuite) TestGetUserPoolDescriptionsError() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	v, err := authx.GetUserPoolDescriptions(ctx, cogIdp)
	s.Assert().Error(err)
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestGetUserPoolByNameError() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	v, err := authx.GetUserPoolByName(ctx, cogIdp, "test-user-pool1")
	s.Assert().Error(err)
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestGetUserPoolByNameError2() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.On("ListUserPools", ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool1")},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		nil, errors.New("test error")).Once()
	v, err := authx.GetUserPoolByName(ctx, cogIdp, "test-user-pool1")
	s.Assert().Error(err)
	s.Assert().Equal("test error", err.Error())
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestGetUserPoolByNameEmpty() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{}, nil)
	v, err := authx.GetUserPoolByName(ctx, cogIdp, "test-user-pool1")
	s.Assert().Error(err)
	s.Assert().Equal("no user pool found", err.Error())
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestGetUserPoolByNameNoMatch() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool2")},
				{Name: aws.String("test-user-pool3")},
			}}, nil)
	v, err := authx.GetUserPoolByName(ctx, cogIdp, "test-user-pool1")
	s.Assert().Error(err)
	s.Assert().Equal("no user pool found", err.Error())
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestGetUserPoolByNameTooManyMatches() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.On("ListUserPools", ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool2")},
				{Name: aws.String("test-user-pool1")},
				{Name: aws.String("test-user-pool1")},
				{Name: aws.String("test-user-pool3")},
			}}, nil).Once()

	v, err := authx.GetUserPoolByName(ctx, cogIdp, "test-user-pool1")
	s.Assert().Error(err)
	s.Assert().Equal("more than one user pool found with name: test-user-pool1", err.Error())
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestGetUserPoolByName() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Name: aws.String("test-user-pool2")},
				{
					Name: aws.String("test-user-pool1"),
					Id:   aws.String("pool1"),
				},
				{Name: aws.String("test-user-pool3")},
			}}, nil)
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-user-pool1"),
				Id:   aws.String("pool1"),
			},
		}, nil).Once()
	v, err := authx.GetUserPoolByName(ctx, cogIdp, "test-user-pool1")
	s.Assert().NoError(err)
	s.Assert().Equal(*v.Name, "test-user-pool1")
	s.Assert().Equal(*v.Id, "pool1")
}

func (s *UserPoolTestSuite) TestGetUserPoolByID() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-user-pool1"),
				Id:   aws.String("test-user-pool-id1"),
			},
		}, nil).Once()
	v, err := authx.GetUserPoolByID(ctx, cogIdp, "test-user-pool-id1")
	s.Assert().NoError(err)
	s.Assert().Equal(*v.Name, "test-user-pool1")
	s.Assert().Equal(*v.Id, "test-user-pool-id1")
}

func (s *UserPoolTestSuite) TestGetUserPoolByNameOrID() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-user-pool1"),
				Id:   aws.String("test-user-pool-id1"),
			},
		}, nil).Times(2)
	v, err := authx.GetUserPoolByNameOrID(ctx, cogIdp, "test-user-pool1")
	s.Assert().NoError(err)
	s.Assert().Equal(*v.Name, "test-user-pool1")
	s.Assert().Equal(*v.Id, "test-user-pool-id1")
	v2, err := authx.GetUserPoolByNameOrID(ctx, cogIdp, "test-user-pool-id1")
	s.Assert().NoError(err)
	s.Assert().Equal(*v2.Name, "test-user-pool1")
	s.Assert().Equal(*v2.Id, "test-user-pool-id1")
}

func (s *UserPoolTestSuite) TestGetUsersInUserPool() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-user-pool1"),
				Id:   aws.String("test-user-pool-id1"),
			},
		}, nil).Once()
	cogIdp.EXPECT().ListUsers(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUsersOutput{
			Users: []types.UserType{
				{Username: aws.String("test-user1")},
				{Username: aws.String("test-user2")},
			}}, nil).Once()
	v, err := authx.GetUsersInUserPool(ctx, cogIdp, "test-user-pool-id1")
	s.Assert().NoError(err)
	s.Assert().NotEmpty(v)
	s.Assert().Len(v, 2)
	s.Assert().Equal("test-user1", *v[0].Username)
	s.Assert().Equal("test-user2", *v[1].Username)
}

func (s *UserPoolTestSuite) TestGetUsersInUserPoolError() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-user-pool1"),
				Id:   aws.String("test-user-pool-id1"),
			},
		}, nil).Once()
	cogIdp.EXPECT().ListUsers(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	v, err := authx.GetUsersInUserPool(ctx, cogIdp, "test-user-pool-id1")
	s.Assert().Error(err)
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestGetUsersInUserPoolError2() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{
					Name: aws.String("test-user-pool1"),
					Id:   aws.String("test-user-pool-id1"),
				},
			}}, nil)
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	v, err := authx.GetUsersInUserPool(ctx, cogIdp, "test-user-pool-id1")
	s.Assert().Error(err)
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestGetUsersInUserPoolError3() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	v, err := authx.GetUsersInUserPool(ctx, cogIdp, "test-user-pool-id1")
	s.Assert().Error(err)
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestCreateUserPool() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-default-user-pool1"),
				Id:   aws.String("test-default-user-pool-id1"),
			},
		}, nil).Once()
	cogIdp.EXPECT().DescribeUserPoolDomain(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolDomainOutput{
			DomainDescription: &types.DomainDescriptionType{
				Domain: aws.String("test-default-user-pool-domain"),
				CustomDomainConfig: &types.CustomDomainConfigType{
					CertificateArn: aws.String("test-default-certificate-arn"),
				},
			},
		}, nil).Once()
	cogIdp.EXPECT().CreateUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.CreateUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-user-pool1"),
				Id:   aws.String("test-user-pool-id1"),
			},
		}, nil).Once()
	cogIdp.EXPECT().CreateUserPoolDomain(ctx, mock.Anything).Return(
		&cognitoidentityprovider.CreateUserPoolDomainOutput{}, nil).Once()
	cogIdp.EXPECT().CreateUserPoolClient(ctx, mock.Anything).Return(
		&cognitoidentityprovider.CreateUserPoolClientOutput{}, nil).Once()
	v, err := authx.CreateUserPool(
		ctx, cogIdp, "test-root-user-pool-id", "test-user-pool1")
	s.Assert().NoError(err)
	s.Assert().Equal(*v.Name, "test-user-pool1")
	s.Assert().Equal(*v.Id, "test-user-pool-id1")
}

func (s *UserPoolTestSuite) TestCreateUserPoolError() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-default-user-pool1"),
				Id:   aws.String("test-default-user-pool-id1"),
			},
		}, nil).Once()
	cogIdp.EXPECT().DescribeUserPoolDomain(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolDomainOutput{
			DomainDescription: &types.DomainDescriptionType{
				Domain: aws.String("test-default-user-pool-domain"),
				CustomDomainConfig: &types.CustomDomainConfigType{
					CertificateArn: aws.String("test-default-certificate-arn"),
				},
			},
		}, nil).Once()
	cogIdp.EXPECT().CreateUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.CreateUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-user-pool1"),
				Id:   aws.String("test-user-pool-id1"),
			},
		}, nil).Once()
	cogIdp.EXPECT().CreateUserPoolDomain(ctx, mock.Anything).Return(
		&cognitoidentityprovider.CreateUserPoolDomainOutput{}, nil).Once()
	cogIdp.EXPECT().CreateUserPoolClient(ctx, mock.Anything).Return(
		nil, errors.New("test error")).Once()
	cogIdp.EXPECT().DeleteUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DeleteUserPoolOutput{}, nil).Once()
	v, err := authx.CreateUserPool(
		ctx, cogIdp, "test-root-user-pool-id", "test-user-pool1")
	s.Assert().Error(err)
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestCreateUserPoolError2() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-default-user-pool1"),
				Id:   aws.String("test-default-user-pool-id1"),
			},
		}, nil).Once()
	cogIdp.EXPECT().DescribeUserPoolDomain(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolDomainOutput{
			DomainDescription: &types.DomainDescriptionType{
				Domain: aws.String("test-default-user-pool-domain"),
				CustomDomainConfig: &types.CustomDomainConfigType{
					CertificateArn: aws.String("test-default-certificate-arn"),
				},
			},
		}, nil).Once()
	cogIdp.EXPECT().CreateUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.CreateUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-user-pool1"),
				Id:   aws.String("test-user-pool-id1"),
			},
		}, nil).Once()
	cogIdp.EXPECT().CreateUserPoolDomain(ctx, mock.Anything).Return(
		&cognitoidentityprovider.CreateUserPoolDomainOutput{}, nil).Once()
	cogIdp.EXPECT().CreateUserPoolClient(ctx, mock.Anything).Return(
		nil, errors.New("test error")).Once()
	cogIdp.EXPECT().DeleteUserPool(ctx, mock.Anything).Return(
		nil, errors.New("delete user pool error")).Once()
	v, err := authx.CreateUserPool(
		ctx, cogIdp, "test-root-user-pool-id", "test-user-pool1")
	s.Assert().Error(err)
	s.Assert().Equal("test error", err.Error())
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestCreateUserPoolError3() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().CreateUserPool(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	v, err := authx.CreateUserPool(
		ctx, cogIdp, "test-root-user-pool-id", "test-user-pool1")
	s.Assert().Error(err)
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestDeleteUserPool() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-user-pool1"),
				Id:   aws.String("test-user-pool-id1"),
			},
		}, nil).Once()
	cogIdp.EXPECT().DeleteUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DeleteUserPoolOutput{}, nil).Once()
	v, err := authx.DeleteUserPool(ctx, cogIdp, "test-user-pool1")
	s.Assert().NoError(err)
	s.Assert().NotNil(v)
}

func (s *UserPoolTestSuite) TestDeleteUserPoolError() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	restore := appenv.OverrideLocalStack(false)
	v, err := authx.DeleteUserPool(ctx, cogIdp, "test-user-pool1")
	restore()
	s.Assert().Error(err)
	s.Assert().Equal("cannot delete user pool in non-localstack environment", err.Error())
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestDeleteUserPoolError2() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		nil, errors.New("test error")).Once()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		nil, errors.New("test error")).Once()
	v, err := authx.DeleteUserPool(ctx, cogIdp, "test-user-pool1")
	s.Assert().Error(err)
	s.Assert().Equal("test error", err.Error())
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestDeleteUserPoolError3() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		nil, errors.New("test error")).Once()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{
					Name: aws.String("test-user-pool1"),
					Id:   aws.String("test-user-pool-id1"),
				},
			}}, nil).Once()
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &types.UserPoolType{
				Name: aws.String("test-user-pool1"),
				Id:   aws.String("test-user-pool-id1"),
			},
		}, nil).Once()
	cogIdp.EXPECT().DeleteUserPool(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	v, err := authx.DeleteUserPool(ctx, cogIdp, "test-user-pool1")
	s.Assert().Error(err)
	s.Assert().Equal("test error", err.Error())
	s.Assert().Nil(v)
}

func (s *UserPoolTestSuite) TestDeleteAllUserPoolsNone() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{}, nil)
	err := authx.DeleteAllUserPools(ctx, cogIdp)
	s.Assert().NoError(err)
}

func (s *UserPoolTestSuite) TestDeleteAllUserPools() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Id: aws.String("test-user-pool-id1")},
				{Id: aws.String("test-user-pool-id2")},
				{Id: aws.String("test-user-pool-id3")},
			}}, nil).Once()
	cogIdp.EXPECT().DeleteUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DeleteUserPoolOutput{}, nil).Times(3)
	err := authx.DeleteAllUserPools(ctx, cogIdp)
	s.Assert().NoError(err)
}

func (s *UserPoolTestSuite) TestDeleteAllUserPoolsPaged() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			NextToken: aws.String("n"),
			UserPools: []types.UserPoolDescriptionType{
				{Id: aws.String("test-user-pool-id1")},
			}}, nil).Once()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Id: aws.String("test-user-pool-id2")},
			}}, nil).Once()
	cogIdp.EXPECT().DeleteUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DeleteUserPoolOutput{}, nil).Times(2)
	err := authx.DeleteAllUserPools(ctx, cogIdp)
	s.Assert().NoError(err)
}

func (s *UserPoolTestSuite) TestDeleteAllUserPoolsError() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		&cognitoidentityprovider.ListUserPoolsOutput{
			UserPools: []types.UserPoolDescriptionType{
				{Id: aws.String("test-user-pool-id1")},
				{Id: aws.String("test-user-pool-id2")},
				{Id: aws.String("test-user-pool-id3")},
			}}, nil).Once()
	cogIdp.EXPECT().DeleteUserPool(ctx, mock.Anything).Return(
		nil, errors.New("test error")).Once()
	err := authx.DeleteAllUserPools(ctx, cogIdp)
	s.Assert().Error(err)
	s.Assert().Equal("test error", err.Error())
}

func (s *UserPoolTestSuite) TestDeleteAllUserPoolsError2() {
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	ctx := context.Background()
	cogIdp.EXPECT().ListUserPools(ctx, mock.Anything).Return(
		nil, errors.New("test error"))
	err := authx.DeleteAllUserPools(ctx, cogIdp)
	s.Assert().Error(err)
	s.Assert().Equal("test error", err.Error())
}
