package authx

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"

	"github.com/apresdev/landlord/internal/appenv"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
)

func CreateIdentityPool(ctx context.Context, cogId AWSCognitoIdentity, name string) error {
	args := cognitoidentity.CreateIdentityPoolInput{
		AllowUnauthenticatedIdentities: true,
		IdentityPoolName:               &name,
	}
	_, err := cogId.CreateIdentityPool(ctx, &args)
	return err
}

func GetIdentityPoolByNameOrID(
	ctx context.Context,
	cogId AWSCognitoIdentity,
	idOrName string) (*types.IdentityPoolShortDescription, error) {

	maxResults := MaxPagingResults
	nextToken := newNextToken()
	args := cognitoidentity.ListIdentityPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	for {
		pools, err := cogId.ListIdentityPools(ctx, &args)
		if err != nil {
			return nil, err
		}
		for _, pool := range pools.IdentityPools {
			if *pool.IdentityPoolId == idOrName || *pool.IdentityPoolName == idOrName {
				return &pool, nil
			}
		}
		if pools.NextToken == nil {
			break
		}
		nextToken = *pools.NextToken
	}
	return nil, errors.New("identity pool not found")
}

func GetIdentityPoolShortDescriptions(ctx context.Context, cogId AWSCognitoIdentity) (
	[]types.IdentityPoolShortDescription, error) {

	maxResults := MaxPagingResults
	nextToken := newNextToken()
	args := cognitoidentity.ListIdentityPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	result := make([]types.IdentityPoolShortDescription, 0)
	for {
		pools, err := cogId.ListIdentityPools(ctx, &args)
		if err != nil {
			return nil, err
		}
		result = append(result, pools.IdentityPools...)
		if pools.NextToken == nil {
			break
		}
		nextToken = *pools.NextToken
		if len(result) >= MaxTotalResults {
			return nil, fmt.Errorf(
				"identity pool limit exceeded: %d >= %d", len(result), MaxTotalResults)
		}
	}
	return result, nil
}

func DeleteIdentityPool(ctx context.Context, cogId AWSCognitoIdentity, id string) error {
	_, err := cogId.DeleteIdentityPool(ctx, &cognitoidentity.DeleteIdentityPoolInput{
		IdentityPoolId: &id})
	return err
}

func DeleteAllIdentityPools(
	ctx context.Context,
	cogId AWSCognitoIdentity) error {

	if !appenv.LocalStack {
		return errors.New("only available in a LocalStack environment")
	}

	maxResults := MaxPagingResults
	nextToken := newNextToken()
	args := cognitoidentity.ListIdentityPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	for {
		pools, err := cogId.ListIdentityPools(ctx, &args)
		if err != nil {
			return err
		}
		for _, pool := range pools.IdentityPools {
			_, err := cogId.DeleteIdentityPool(
				ctx, &cognitoidentity.DeleteIdentityPoolInput{
					IdentityPoolId: pool.IdentityPoolId})
			if err != nil {
				return err
			}
		}
		if pools.NextToken == nil {
			break
		}
		nextToken = *pools.NextToken
	}
	return nil
}

// ExtractUserPoolID extracts the user pool ID from the given ARN
func ExtractUserPoolID(arn string) (string, string, string, error) {
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return "", "", "", fmt.Errorf("invalid ARN format")
	}

	region := parts[3]
	accountID := parts[4]

	// userpool/<user-pool-id>
	userPoolParts := strings.Split(parts[5], "/")
	if len(userPoolParts) != 2 {
		return "", "", "", fmt.Errorf("invalid user pool ARN format")
	}

	userPoolID := userPoolParts[1]
	return region, accountID, userPoolID, nil
}

// ConvertToProviderName converts a user pool ARN to an identity pool provider name
func ConvertToProviderName(arn string) (string, error) {
	region, _, userPoolID, err := ExtractUserPoolID(arn)
	if err != nil {
		return "", err
	}

	providerName := fmt.Sprintf("cognito-idp.%s.amazonaws.com/%s", region, userPoolID)
	return providerName, nil
}

func AssociateUserPool(
	ctx context.Context,
	cogId AWSCognitoIdentity,
	cogIdp AWSCognitoIdentityProvider,
	idPoolID string,
	userPoolID string) error {

	idPoolShort, err := GetIdentityPoolByNameOrID(ctx, cogId, idPoolID)
	if err != nil {
		return err
	}
	pool, err := cogId.DescribeIdentityPool(ctx,
		&cognitoidentity.DescribeIdentityPoolInput{
			IdentityPoolId: idPoolShort.IdentityPoolId})
	if err != nil {
		return err
	}
	userPool, err := GetUserPoolByNameOrID(ctx, cogIdp, userPoolID)
	if err != nil {
		return err
	}
	providerName, err := ConvertToProviderName(*userPool.Arn)
	if err != nil {
		return err
	}
	providers := pool.CognitoIdentityProviders
	if providers == nil {
		providers = make([]types.CognitoIdentityProvider, 0)
	}
	for _, provider := range providers {
		if *provider.ProviderName == providerName {
			return fmt.Errorf(
				"user pool '%s' already associated with identity pool '%s'",
				userPoolID, idPoolID)
		}
	}
	userPoolClients, err := cogIdp.ListUserPoolClients(ctx,
		&cognitoidentityprovider.ListUserPoolClientsInput{
			UserPoolId: userPool.Id})
	if err != nil {
		return err
	}
	if len(userPoolClients.UserPoolClients) == 0 {
		return fmt.Errorf("no clients found for user pool '%s'", userPoolID)
	}
	if len(userPoolClients.UserPoolClients) > 1 {
		return fmt.Errorf("more than one client found for user pool '%s'", userPoolID)
	}
	providers = append(providers, types.CognitoIdentityProvider{
		ClientId:     userPoolClients.UserPoolClients[0].ClientId,
		ProviderName: aws.String(providerName),
	})
	args := cognitoidentity.UpdateIdentityPoolInput{
		AllowUnauthenticatedIdentities: pool.AllowUnauthenticatedIdentities,
		IdentityPoolId:                 pool.IdentityPoolId,
		IdentityPoolName:               pool.IdentityPoolName,
		CognitoIdentityProviders:       providers,
	}
	_, err = cogId.UpdateIdentityPool(ctx, &args)
	return err
}
