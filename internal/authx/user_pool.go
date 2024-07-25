package authx

import (
	"context"
	"errors"
	"fmt"

	"github.com/sethvargo/go-password/password"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"go.uber.org/zap"
)

// GetUserPoolDescriptions returns the descriptions of all user pools
// in the AWS Cognito Identity Provider for the current AWS_PROFILE
func GetUserPoolDescriptions(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider) (
	[]types.UserPoolDescriptionType, error) {

	result := make([]types.UserPoolDescriptionType, 0)
	maxResults := MaxPagingResults
	nextToken := newNextToken()
	args := cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	for {
		u, err := cogIdp.ListUserPools(ctx, &args)
		if err != nil {
			return nil, err
		}
		result = append(result, u.UserPools...)
		if u.NextToken == nil {
			break
		}
		nextToken = *u.NextToken
	}
	return result, nil
}

func getUserPool(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	up types.UserPoolDescriptionType) (*types.UserPoolType, error) {

	args := cognitoidentityprovider.DescribeUserPoolInput{
		UserPoolId: up.Id,
	}
	o, err := cogIdp.DescribeUserPool(ctx, &args)
	if err != nil {
		return nil, err
	}
	return o.UserPool, nil
}

func getUserPoolDescriptions(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	match func(descriptionType types.UserPoolDescriptionType) bool) (
	[]types.UserPoolDescriptionType, error) {

	maxResults := MaxPagingResults
	nextToken := newNextToken()
	args := cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	result := make([]types.UserPoolDescriptionType, 0)
	for {
		u, err := cogIdp.ListUserPools(ctx, &args)
		if err != nil {
			return nil, err
		}
		for _, up := range u.UserPools {
			if match(up) {
				result = append(result, up)
			}
		}
		if u.NextToken == nil {
			break
		}
	}
	return result, nil
}

func GetUserPoolByName(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolName string) (*types.UserPoolType, error) {

	ups, err := getUserPoolDescriptions(
		ctx, cogIdp, func(descriptionType types.UserPoolDescriptionType) bool {
			return descriptionType.Name != nil && *descriptionType.Name == userPoolName
		})
	if err != nil {
		return nil, err
	}
	if len(ups) == 0 {
		return nil, fmt.Errorf("no user pool found")
	}
	if len(ups) > 1 {
		return nil, fmt.Errorf("more than one user pool found with name: %s", userPoolName)
	}
	return getUserPool(ctx, cogIdp, ups[0])
}

func GetUserPoolByID(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string) (*types.UserPoolType, error) {

	up, err := getUserPool(ctx, cogIdp, types.UserPoolDescriptionType{Id: &userPoolID})
	if err != nil {
		return nil, err
	}
	return up, nil
}

func GetUserPoolByNameOrID(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	nameOrID string) (*types.UserPoolType, error) {

	if nameOrID == "" {
		return nil, errors.New("user pool name or ID must be set")
	}
	up, err := GetUserPoolByID(ctx, cogIdp, nameOrID)
	if err == nil {
		return up, nil
	}
	return GetUserPoolByName(ctx, cogIdp, nameOrID)
}

func GetUsersInUserPool(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPool string) ([]types.UserType, error) {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, userPool)
	if err != nil {
		return nil, err
	}
	args := cognitoidentityprovider.ListUsersInput{
		UserPoolId: up.Id,
	}
	o, err := cogIdp.ListUsers(ctx, &args)
	if err != nil {
		return nil, err
	}
	return o.Users, nil
}

func CreateUserPool(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	rootUserPoolID string,
	userPoolName string) (*types.UserPoolType, error) {

	args := cognitoidentityprovider.CreateUserPoolInput{
		PoolName: aws.String(userPoolName),
		UsernameConfiguration: &types.UsernameConfigurationType{
			CaseSensitive: aws.Bool(false),
		},
	}
	o, err := cogIdp.CreateUserPool(ctx, &args)
	if err != nil {
		return nil, err
	}
	defaultUP, err := cogIdp.DescribeUserPool(ctx,
		&cognitoidentityprovider.DescribeUserPoolInput{
			UserPoolId: aws.String(rootUserPoolID),
		})
	if err != nil {
		return nil, err
	}
	defaultUPDomain, err := cogIdp.DescribeUserPoolDomain(ctx,
		&cognitoidentityprovider.DescribeUserPoolDomainInput{
			Domain: defaultUP.UserPool.CustomDomain,
		})
	if err != nil {
		return nil, err
	}
	if defaultUPDomain.DomainDescription == nil {
		return nil, errors.New("no domain description found")
	}
	if defaultUPDomain.DomainDescription.CustomDomainConfig == nil {
		return nil, errors.New("no custom domain config found")
	}
	if defaultUPDomain.DomainDescription.CustomDomainConfig.CertificateArn == nil {
		return nil, errors.New("no certificate ARN found")
	}
	_, err = cogIdp.CreateUserPoolDomain(
		ctx, &cognitoidentityprovider.CreateUserPoolDomainInput{
			Domain:     defaultUP.UserPool.CustomDomain,
			UserPoolId: o.UserPool.Id,
			CustomDomainConfig: &types.CustomDomainConfigType{
				CertificateArn: defaultUPDomain.DomainDescription.CustomDomainConfig.CertificateArn,
			},
		})
	if err != nil {
		return nil, err
	}

	clientName := fmt.Sprintf("%s-client", userPoolName)
	_, err = cogIdp.CreateUserPoolClient(ctx, &cognitoidentityprovider.CreateUserPoolClientInput{
		ClientName: aws.String(clientName),
		UserPoolId: o.UserPool.Id,
		ExplicitAuthFlows: []types.ExplicitAuthFlowsType{
			types.ExplicitAuthFlowsTypeAllowUserPasswordAuth,
			types.ExplicitAuthFlowsTypeAllowRefreshTokenAuth,
			types.ExplicitAuthFlowsTypeAllowUserSrpAuth,
		},
	})
	if err != nil {
		var err2 error
		_, err2 = cogIdp.DeleteUserPool(
			ctx,
			&cognitoidentityprovider.DeleteUserPoolInput{UserPoolId: o.UserPool.Id})
		if err2 != nil {
			zap.L().Error("failed to delete user pool", zap.Error(err2))
		}
		return nil, err
	}
	return o.UserPool, nil
}

func DeleteUserPool(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPool string) (*types.UserPoolType, error) {

	if !appenv.LocalStack {
		return nil, errors.New("cannot delete user pool in non-localstack environment")
	}

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, userPool)
	if err != nil {
		return nil, err
	}
	args := cognitoidentityprovider.DeleteUserPoolInput{
		UserPoolId: up.Id,
	}
	_, err = cogIdp.DeleteUserPool(ctx, &args)
	if err != nil {
		return nil, err
	}
	return up, nil
}

func DeleteAllUserPools(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider) error {

	maxResults := MaxPagingResults
	nextToken := newNextToken()
	args := cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	for {
		u, err := cogIdp.ListUserPools(ctx, &args)
		if err != nil {
			return err
		}
		for _, up := range u.UserPools {
			_, err := cogIdp.DeleteUserPool(
				ctx, &cognitoidentityprovider.DeleteUserPoolInput{
					UserPoolId: up.Id,
				})
			if err != nil {
				return err
			}
		}
		if u.NextToken == nil {
			break
		}
		nextToken = *u.NextToken
	}
	return nil
}

func DeleteUserFromUserPool(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPool string,
	email string) error {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, userPool)
	if err != nil {
		return err
	}
	args := cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: up.Id,
		Username:   aws.String(email),
	}
	_, err = cogIdp.AdminDeleteUser(ctx, &args)
	if err != nil {
		return err
	}
	return nil
}

func CreateUser(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPool string,
	username string) error {

	pwd, err := password.Generate(16, 4, 4, false, false)
	if err != nil {
		return err
	}

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, userPool)
	if err != nil {
		return err
	}
	_, err = cogIdp.AdminCreateUser(ctx,
		&cognitoidentityprovider.AdminCreateUserInput{
			UserPoolId:        up.Id,
			Username:          &username,
			TemporaryPassword: &pwd,
		})
	return err
}

func DeleteUser(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPool, email string) error {

	up, err := GetUserPoolByNameOrID(ctx, cogIdp, userPool)
	if err != nil {
		return err
	}
	_, err = cogIdp.AdminDeleteUser(ctx,
		&cognitoidentityprovider.AdminDeleteUserInput{
			UserPoolId: up.Id,
			Username:   &email,
		})
	return err
}
