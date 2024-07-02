package authx

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/aws/aws-sdk-go-v2/service/ses"

	"github.com/aws/aws-sdk-go-v2/service/iam"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
)

//go:generate mockery --output ../mocks --name AWSSTS --with-expecter=true
type AWSSTS interface {
	GetCallerIdentity(
		ctx context.Context,
		params *sts.GetCallerIdentityInput,
		optFns ...func(*sts.Options)) (
		*sts.GetCallerIdentityOutput, error)
}

//go:generate mockery --output ../mocks --name AWSSES --with-expecter=true
type AWSSES interface {
	SendEmail(
		ctx context.Context,
		params *ses.SendEmailInput,
		optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
}

//go:generate mockery --output ../mocks --name AWSIAM --with-expecter=true
type AWSIAM interface {
	CreateRole(
		ctx context.Context,
		params *iam.CreateRoleInput,
		optFns ...func(*iam.Options)) (
		*iam.CreateRoleOutput, error)
	DeleteRole(
		ctx context.Context,
		params *iam.DeleteRoleInput,
		optFns ...func(*iam.Options)) (
		*iam.DeleteRoleOutput, error)
	ListRoles(
		ctx context.Context,
		params *iam.ListRolesInput,
		optFns ...func(*iam.Options)) (
		*iam.ListRolesOutput, error)
}

//go:generate mockery --output ../mocks --name AWSCognitoIdentity --with-expecter=true
type AWSCognitoIdentity interface {
	CreateIdentityPool(
		ctx context.Context,
		params *cognitoidentity.CreateIdentityPoolInput,
		optFns ...func(*cognitoidentity.Options)) (
		*cognitoidentity.CreateIdentityPoolOutput, error)
	DescribeIdentityPool(
		ctx context.Context,
		params *cognitoidentity.DescribeIdentityPoolInput,
		optFns ...func(*cognitoidentity.Options)) (
		*cognitoidentity.DescribeIdentityPoolOutput, error)
	DeleteIdentityPool(
		ctx context.Context,
		params *cognitoidentity.DeleteIdentityPoolInput,
		optFns ...func(*cognitoidentity.Options)) (
		*cognitoidentity.DeleteIdentityPoolOutput, error)
	ListIdentityPools(
		ctx context.Context,
		params *cognitoidentity.ListIdentityPoolsInput,
		optFns ...func(*cognitoidentity.Options)) (
		*cognitoidentity.ListIdentityPoolsOutput, error)
	UpdateIdentityPool(
		ctx context.Context,
		params *cognitoidentity.UpdateIdentityPoolInput,
		optFns ...func(*cognitoidentity.Options)) (
		*cognitoidentity.UpdateIdentityPoolOutput, error)
}

//go:generate mockery --output ../mocks --name AWSCognitoIdentityProvider --with-expecter=true
type AWSCognitoIdentityProvider interface {
	InitiateAuth(
		ctx context.Context,
		params *cognitoidentityprovider.InitiateAuthInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.InitiateAuthOutput, error)
	CreateUserPool(
		ctx context.Context,
		params *cognitoidentityprovider.CreateUserPoolInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.CreateUserPoolOutput, error)
	CreateUserPoolClient(
		ctx context.Context,
		params *cognitoidentityprovider.CreateUserPoolClientInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.CreateUserPoolClientOutput, error)
	CreateUserPoolDomain(
		ctx context.Context,
		params *cognitoidentityprovider.CreateUserPoolDomainInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.CreateUserPoolDomainOutput, error)
	DeleteUserPool(
		ctx context.Context,
		params *cognitoidentityprovider.DeleteUserPoolInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.DeleteUserPoolOutput, error)
	ListUserPools(
		ctx context.Context,
		params *cognitoidentityprovider.ListUserPoolsInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.ListUserPoolsOutput, error)
	DescribeUserPool(
		ctx context.Context,
		params *cognitoidentityprovider.DescribeUserPoolInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.DescribeUserPoolOutput, error)
	DescribeUserPoolDomain(
		ctx context.Context,
		params *cognitoidentityprovider.DescribeUserPoolDomainInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.DescribeUserPoolDomainOutput, error)
	ListUserPoolClients(
		ctx context.Context,
		params *cognitoidentityprovider.ListUserPoolClientsInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.ListUserPoolClientsOutput, error)
	DescribeUserPoolClient(
		ctx context.Context,
		params *cognitoidentityprovider.DescribeUserPoolClientInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.DescribeUserPoolClientOutput, error)
	ListUsers(
		ctx context.Context,
		params *cognitoidentityprovider.ListUsersInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.ListUsersOutput, error)
	AdminConfirmSignUp(
		ctx context.Context,
		params *cognitoidentityprovider.AdminConfirmSignUpInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminConfirmSignUpOutput, error)
	AdminCreateUser(
		ctx context.Context,
		params *cognitoidentityprovider.AdminCreateUserInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminCreateUserOutput, error)
	AdminDeleteUser(
		ctx context.Context,
		params *cognitoidentityprovider.AdminDeleteUserInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminDeleteUserOutput, error)
	AdminSetUserPassword(
		ctx context.Context,
		params *cognitoidentityprovider.AdminSetUserPasswordInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminSetUserPasswordOutput, error)
	AdminUpdateUserAttributes(
		ctx context.Context,
		params *cognitoidentityprovider.AdminUpdateUserAttributesInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminUpdateUserAttributesOutput, error)
	SignUp(
		ctx context.Context,
		params *cognitoidentityprovider.SignUpInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.SignUpOutput, error)
	ConfirmSignUp(
		ctx context.Context,
		params *cognitoidentityprovider.ConfirmSignUpInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.ConfirmSignUpOutput, error)
}
