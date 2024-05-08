package authx

import (
	"context"

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

func GetIdentityPoolShortDescriptions(ctx context.Context, cogId AWSCognitoIdentity) (
	[]types.IdentityPoolShortDescription, error) {

	// TODO: actually process through the pagination
	// TODO: limit the number of results if they are excessive
	maxResults := int32(10)
	nextToken := newNextToken()
	args := cognitoidentity.ListIdentityPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	o, err := cogId.ListIdentityPools(ctx, &args)
	if err != nil {
		return nil, err
	}
	return o.IdentityPools, nil
}

func DeleteAllIdentityPools(
	ctx context.Context,
	cogId AWSCognitoIdentity) error {

	maxResults := int32(MaxPagingResults)
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
