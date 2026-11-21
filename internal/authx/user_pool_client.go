package authx

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func getUserPoolClients(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string) ([]types.UserPoolClientDescription, error) {

	args := cognitoidentityprovider.ListUserPoolClientsInput{
		UserPoolId: aws.String(userPoolID),
	}
	o, err := cogIdp.ListUserPoolClients(ctx, &args)
	if err != nil {
		return nil, err
	}
	return o.UserPoolClients, nil
}

// selectUserPoolClientDescription picks the client to use from a user pool.
// A pool with exactly one client always resolves to that client (so tenant
// pools need no configuration). A pool with several clients (e.g. one for the
// console and others for machine integrations) requires preferred, the name or
// ID of the client to use, to match exactly one of them.
func selectUserPoolClientDescription(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string,
	preferred string) (
	*types.UserPoolClientDescription, error) {

	c, err := getUserPoolClients(ctx, cogIdp, userPoolID)
	if err != nil {
		return nil, err
	}
	if len(c) == 0 {
		return nil, fmt.Errorf("no user pool clients found")
	}
	if preferred != "" {
		for i := range c {
			if aws.ToString(c[i].ClientId) == preferred ||
				aws.ToString(c[i].ClientName) == preferred {
				return &c[i], nil
			}
		}
	}
	if len(c) == 1 {
		return &c[0], nil
	}
	if preferred == "" {
		return nil, fmt.Errorf(
			"more than one user pool client found and LANDLORD_DEFAULT_USER_POOL_CLIENT is not set")
	}
	return nil, fmt.Errorf(
		"more than one user pool client found and none matches %q", preferred)
}

// GetUserPoolClient finds the user pool client to use for the given user pool.
// See selectUserPoolClientDescription for how preferred (a client name or ID,
// possibly empty) is applied.
func GetUserPoolClient(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string,
	preferred string) (
	*types.UserPoolClientType, error) {

	cd, err := selectUserPoolClientDescription(ctx, cogIdp, userPoolID, preferred)
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
