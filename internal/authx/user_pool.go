package authx

import (
	"context"
	"errors"
	"fmt"

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
	maxResults := int32(MaxPagingResults)
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

	maxResults := int32(MaxPagingResults)
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
		return nil, fmt.Errorf("more than one user pool found")
	}
	return getUserPool(ctx, cogIdp, ups[0])
}

func GetUserPoolByID(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string) (*types.UserPoolType, error) {

	ups, err := getUserPoolDescriptions(
		ctx, cogIdp, func(descriptionType types.UserPoolDescriptionType) bool {
			return descriptionType.Id != nil && *descriptionType.Id == userPoolID
		})
	if err != nil {
		return nil, err
	}
	if len(ups) == 0 {
		return nil, fmt.Errorf("no user pool found")
	}
	if len(ups) > 1 {
		return nil, fmt.Errorf("more than one user pool found")
	}
	return getUserPool(ctx, cogIdp, ups[0])
}

func GetUserPoolByNameOrID(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	nameOrID string) (*types.UserPoolType, error) {

	if nameOrID == "" {
		return nil, errors.New("user pool name or ID must be set")
	}
	up, err := GetUserPoolByName(ctx, cogIdp, nameOrID)
	if err == nil {
		return up, nil
	}
	return GetUserPoolByID(ctx, cogIdp, nameOrID)
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
	name string) (*types.UserPoolType, error) {

	args := cognitoidentityprovider.CreateUserPoolInput{
		PoolName: aws.String(name),
	}
	o, err := cogIdp.CreateUserPool(ctx, &args)
	if err != nil {
		return nil, err
	}
	_, err = cogIdp.CreateUserPoolClient(ctx, &cognitoidentityprovider.CreateUserPoolClientInput{
		ClientName: aws.String(name),
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

	if !appenv.Localstack {
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

	maxResults := int32(MaxPagingResults)
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
