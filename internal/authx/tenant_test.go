package authx_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cogidptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/apresdev/landlord/internal/mocks"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/suite"
)

type TenantTestSuite struct {
	suite.Suite
}

func TestTenantTestSuite(t *testing.T) {
	suite.Run(t, &TenantTestSuite{})
}

func (s *TenantTestSuite) TestCreateTenant() {
	ctx := context.Background()
	stsClient := mocks.NewAWSSTS(s.T())
	stsClient.EXPECT().GetCallerIdentity(ctx, mock.Anything).Return(
		&sts.GetCallerIdentityOutput{Account: aws.String("aws-region")}, nil)
	awsCfg := aws.Config{}
	i := mocks.NewAWSIAM(s.T())
	i.EXPECT().CreateRole(ctx, mock.Anything).Return(
		&iam.CreateRoleOutput{}, nil)
	cogIdp := mocks.NewAWSCognitoIdentityProvider(s.T())
	cogIdp.EXPECT().DescribeUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolOutput{
			UserPool: &cogidptypes.UserPoolType{Id: aws.String("default-user-pool-id")},
		}, nil)
	cogIdp.EXPECT().DescribeUserPoolDomain(ctx, mock.Anything).Return(
		&cognitoidentityprovider.DescribeUserPoolDomainOutput{
			DomainDescription: &cogidptypes.DomainDescriptionType{
				Domain: aws.String("default-user-pool-domain"),
				CustomDomainConfig: &cogidptypes.CustomDomainConfigType{
					CertificateArn: aws.String("default-certificate-arn")}},
		}, nil)
	cogIdp.EXPECT().CreateUserPool(ctx, mock.Anything).Return(
		&cognitoidentityprovider.CreateUserPoolOutput{
			UserPool: &cogidptypes.UserPoolType{Id: aws.String("user-pool-id")},
		}, nil)
	cogIdp.EXPECT().CreateUserPoolClient(ctx, mock.Anything).Return(
		&cognitoidentityprovider.CreateUserPoolClientOutput{
			UserPoolClient: &cogidptypes.UserPoolClientType{
				ClientId: aws.String("user-pool-client-id")},
		}, nil)
	cogIdp.EXPECT().CreateUserPoolDomain(ctx, mock.Anything).Return(
		&cognitoidentityprovider.CreateUserPoolDomainOutput{}, nil)
	tens, err := authx.InitTenants(ctx, awsCfg, stsClient, cogIdp)
	s.Assert().NoError(err)
	s.Assert().NotNil(tens)
	ten, err := tens.CreateTenant(
		ctx, i, cogIdp, "ten1", "ten1-full-name", "auth1.example.dev")
	s.Assert().NoError(err)
	s.Assert().Equal("ten1", ten.Name)
	s.Assert().Equal("ten1-full-name", ten.FullName)
	_, err = tens.CreateTenant(
		ctx, i, cogIdp, "ten1", "ten1-full-name-2", "auth2.example.dev")
	s.Assert().Error(err)
}
