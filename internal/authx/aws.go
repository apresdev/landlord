package authx

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/aws/aws-sdk-go-v2/service/ses"
)

type AWSSES interface {
	SendEmail(
		ctx context.Context,
		params *ses.SendEmailInput,
		optFns ...func(*ses.Options)) (*ses.SendEmailOutput, error)
}

type AWSCognitoIdentityProvider interface {
	InitiateAuth(
		ctx context.Context,
		params *cognitoidentityprovider.InitiateAuthInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.InitiateAuthOutput, error)
	RespondToAuthChallenge(
		ctx context.Context,
		params *cognitoidentityprovider.RespondToAuthChallengeInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.RespondToAuthChallengeOutput, error)
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
	AdminDisableUser(
		ctx context.Context,
		params *cognitoidentityprovider.AdminDisableUserInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminDisableUserOutput, error)
	AdminEnableUser(
		ctx context.Context,
		params *cognitoidentityprovider.AdminEnableUserInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminEnableUserOutput, error)
	AdminGetUser(
		ctx context.Context,
		params *cognitoidentityprovider.AdminGetUserInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminGetUserOutput, error)
	AdminListUserAuthEvents(
		ctx context.Context,
		params *cognitoidentityprovider.AdminListUserAuthEventsInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminListUserAuthEventsOutput, error)
	AdminSetUserPassword(
		ctx context.Context,
		params *cognitoidentityprovider.AdminSetUserPasswordInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminSetUserPasswordOutput, error)
	AdminResetUserPassword(
		ctx context.Context,
		params *cognitoidentityprovider.AdminResetUserPasswordInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminResetUserPasswordOutput, error)
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
	CreateGroup(
		ctx context.Context,
		params *cognitoidentityprovider.CreateGroupInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.CreateGroupOutput, error)
	DeleteGroup(
		ctx context.Context,
		params *cognitoidentityprovider.DeleteGroupInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.DeleteGroupOutput, error)
	GetGroup(
		ctx context.Context,
		params *cognitoidentityprovider.GetGroupInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.GetGroupOutput, error)
	AdminAddUserToGroup(
		ctx context.Context,
		params *cognitoidentityprovider.AdminAddUserToGroupInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminAddUserToGroupOutput, error)
	AdminRemoveUserFromGroup(
		ctx context.Context,
		params *cognitoidentityprovider.AdminRemoveUserFromGroupInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminRemoveUserFromGroupOutput, error)
	ForgotPassword(
		ctx context.Context,
		params *cognitoidentityprovider.ForgotPasswordInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.ForgotPasswordOutput, error)
	ConfirmForgotPassword(
		ctx context.Context,
		params *cognitoidentityprovider.ConfirmForgotPasswordInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.ConfirmForgotPasswordOutput, error)
	AdminSetUserMFAPreference(
		ctx context.Context,
		params *cognitoidentityprovider.AdminSetUserMFAPreferenceInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AdminSetUserMFAPreferenceOutput, error)
	AssociateSoftwareToken(
		ctx context.Context,
		params *cognitoidentityprovider.AssociateSoftwareTokenInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.AssociateSoftwareTokenOutput, error)
	VerifySoftwareToken(
		ctx context.Context,
		params *cognitoidentityprovider.VerifySoftwareTokenInput,
		optFns ...func(*cognitoidentityprovider.Options)) (
		*cognitoidentityprovider.VerifySoftwareTokenOutput, error)
}
