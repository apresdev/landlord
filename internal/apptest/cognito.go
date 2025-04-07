package apptest

import (
	"context"
	"strconv"

	"github.com/apresdev/landlord/internal/tinytools"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"go.uber.org/zap"
)

// dangerousRecreateUserPool optional deletes the user pool with the given name
// and creates a new one with the same name.
//
// This is meant for testing only and only works in a localstack environment.
func dangerousRecreateUserPool(cfg aws.Config, userPoolName string) {
	ctx := context.Background()
	cogIdp := cognitoidentityprovider.NewFromConfig(cfg)
	maxResults := int32(10)
	nextToken := tinytools.NewNextToken()
	args := cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: &maxResults,
		NextToken:  &nextToken,
	}
	result := make([]types.UserPoolDescriptionType, 0)
	for {
		u, err := cogIdp.ListUserPools(ctx, &args)
		if err != nil {
			zap.L().Fatal("failed to list user pools", zap.Error(err))
		}
		for _, up := range u.UserPools {
			if aws.ToString(up.Name) == userPoolName {
				result = append(result, up)
			}
		}
		if u.NextToken == nil {
			break
		}
		args.NextToken = u.NextToken
	}
	zap.L().Info("found user pools", zap.Int("count", len(result)))
	for _, up := range result {
		zap.L().Info("deleting user pool", zap.String("name", aws.ToString(up.Name)))
		_, err := cogIdp.DeleteUserPool(ctx, &cognitoidentityprovider.DeleteUserPoolInput{
			UserPoolId: up.Id,
		})
		if err != nil {
			zap.L().Fatal("failed to delete user pool", zap.Error(err))
		}
	}

	claims := []struct {
		ClaimName string
		MaxLen    int
	}{
		{ClaimName: "tenant", MaxLen: 64},
		{ClaimName: "user", MaxLen: 64},
		{ClaimName: "impersonate_tenant", MaxLen: 64},
		{ClaimName: "impersonate_group", MaxLen: 64},
		{ClaimName: "impersonate_sub", MaxLen: 128},
		{ClaimName: "impersonate_email", MaxLen: 128},
		{ClaimName: "impersonate_user", MaxLen: 128},
	}
	customAttributes := make([]types.SchemaAttributeType, len(claims))
	for i, claim := range claims {
		customAttributes[i] = types.SchemaAttributeType{
			Name:                   aws.String(claim.ClaimName),
			AttributeDataType:      "String",
			DeveloperOnlyAttribute: aws.Bool(false),
			Mutable:                aws.Bool(true),
			StringAttributeConstraints: &types.StringAttributeConstraintsType{
				MinLength: aws.String("0"),
				MaxLength: aws.String(strconv.Itoa(claim.MaxLen)),
			},
		}
	}
	input := &cognitoidentityprovider.CreateUserPoolInput{
		PoolName: aws.String(userPoolName),
		Schema:   customAttributes,
	}
	_, err := cogIdp.CreateUserPool(ctx, input)
	if err != nil {
		zap.L().Fatal("failed to create user pool", zap.Error(err))
	}
	zap.L().Info("created user pool", zap.String("name", userPoolName))
}
