package authx

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func getUserPoolClients(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	up *types.UserPoolType) ([]types.UserPoolClientDescription, error) {

	args := cognitoidentityprovider.ListUserPoolClientsInput{
		UserPoolId: up.Id,
	}
	o, err := cogIdp.ListUserPoolClients(ctx, &args)
	if err != nil {
		return nil, err
	}
	return o.UserPoolClients, nil
}

func getSingletonUserPoolClientDescription(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	up *types.UserPoolType) (
	*types.UserPoolClientDescription, error) {

	c, err := getUserPoolClients(ctx, cogIdp, up)
	if err != nil {
		return nil, err
	}
	if len(c) == 0 {
		return nil, fmt.Errorf("no user pool clients found")
	}
	if len(c) > 1 {
		return nil, fmt.Errorf("more than one user pool client found")
	}
	return &c[0], nil
}

func GetSingletonUserPoolClient(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	up *types.UserPoolType) (
	*types.UserPoolClientType, error) {

	cd, err := getSingletonUserPoolClientDescription(ctx, cogIdp, up)
	if err != nil {
		return nil, err
	}
	args := cognitoidentityprovider.DescribeUserPoolClientInput{
		ClientId:   cd.ClientId,
		UserPoolId: cd.UserPoolId,
	}
	o, err := cogIdp.DescribeUserPoolClient(ctx, &args)
	if err != nil {
		return nil, err
	}
	return o.UserPoolClient, nil
}
