package authx

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

const defaultMaxResults = int32(10)

// GetUserPoolDescriptions returns the descriptions of all user pools
// in the AWS Cognito Identity Provider for the current AWS_PROFILE
func GetUserPoolDescriptions(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider) (
	[]types.UserPoolDescriptionType, error) {

	result := make([]types.UserPoolDescriptionType, 0)
	maxResults := defaultMaxResults
	args := cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: &maxResults,
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
		args.NextToken = u.NextToken
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

	maxResults := defaultMaxResults
	args := cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: &maxResults,
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
		args.NextToken = u.NextToken
	}
	return result, nil
}

// GetUserPoolByName returns the user pool with the given name
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

// GetUserPoolByID returns the user pool with the given ID
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

// GetUserPoolByNameOrID returns the user pool with the given name or ID
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

// GetUsersInUserPool returns the users in the user pool with the given name or ID
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

// DeleteUserFromUserPool deletes the user with the given email from the user pool
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

// CognitoUserPoolURL returns the URL to the user pool in the AWS console
func CognitoUserPoolURL(region string, userPoolID string) string {
	return fmt.Sprintf(
		"https://%s.console.aws.amazon.com/cognito/v2/idp/user-pools/%s",
		region, userPoolID)
}

// CognitoUserDetailsURL returns the URL to the user details in the AWS console
func CognitoUserDetailsURL(region string, userPoolID string, userID string) string {
	return fmt.Sprintf(
		"https://%s.console.aws.amazon.com/cognito/v2/idp/user-pools/%s/users/details/%s",
		region, userPoolID, userID)
}
