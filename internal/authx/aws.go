package authx

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
)

//go:generate mockery --output ../mocks --name AWSCognitoIdentity --with-expecter=true
type AWSCognitoIdentity interface {
	CreateIdentityPool(
		ctx context.Context,
		params *cognitoidentity.CreateIdentityPoolInput,
		optFns ...func(*cognitoidentity.Options)) (
		*cognitoidentity.CreateIdentityPoolOutput, error)
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
	AdminDeleteUser(
		ctx context.Context,
		params *cognitoidentityprovider.AdminDeleteUserInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminDeleteUserOutput, error)
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
