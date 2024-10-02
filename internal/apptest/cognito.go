package apptest

import (
	"context"
	"os"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/localstack"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"go.uber.org/zap"
)

// DangerousRecreateUserPool optional deletes the user pool with the name
// LANDLORD_DEFAULT_USER_POOL, and creates a new one with the same name.
// A function that is meant for testing only.
func DangerousRecreateUserPool() {
	if os.Getenv("AWS_PROFILE") != "localstack" {
		panic("AWS_PROFILE must be set to 'localstack'")
	}
	if !localstack.IsRunning() {
		panic("localstack not running")
	}
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		zap.L().Fatal("unable to load SDK config", zap.Error(err))
	}

	cogIdp := cognitoidentityprovider.NewFromConfig(cfg)
	maxResults := int32(10)
	nextToken := authx.NewNextToken()
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
			if aws.ToString(up.Name) == os.Getenv("LANDLORD_DEFAULT_USER_POOL") {
				result = append(result, up)
			}
		}
		if u.NextToken == nil {
			break
		}
	}
	if len(result) > 1 {
		zap.L().Fatal("found more than one user pool with the same name")
	}
	if len(result) == 0 {
		zap.L().Info("no user pool found, creating a new one")
	} else {
		zap.L().Info("user pool found, deleting it")
		_, err = cogIdp.DeleteUserPool(ctx, &cognitoidentityprovider.DeleteUserPoolInput{
			UserPoolId: result[0].Id,
		})
		if err != nil {
			zap.L().Fatal("failed to delete user pool", zap.Error(err))
		}
	}
	customAttributes := []types.SchemaAttributeType{
		{
			Name:                   aws.String("impersonate_group"),
			AttributeDataType:      "String",
			DeveloperOnlyAttribute: aws.Bool(false),
			Mutable:                aws.Bool(true),
			StringAttributeConstraints: &types.StringAttributeConstraintsType{
				MinLength: aws.String("0"),
				MaxLength: aws.String("64"),
			},
		},
		{
			Name:                   aws.String("impersonate_user"),
			AttributeDataType:      "String",
			DeveloperOnlyAttribute: aws.Bool(false),
			Mutable:                aws.Bool(true),
			StringAttributeConstraints: &types.StringAttributeConstraintsType{
				MinLength: aws.String("0"),
				MaxLength: aws.String("128"),
			},
		},
		{
			Name:                   aws.String("tenant"),
			AttributeDataType:      "String",
			DeveloperOnlyAttribute: aws.Bool(false),
			Mutable:                aws.Bool(true),
			StringAttributeConstraints: &types.StringAttributeConstraintsType{
				MinLength: aws.String("0"),
				MaxLength: aws.String("64"),
			},
		},
	}
	input := &cognitoidentityprovider.CreateUserPoolInput{
		PoolName: aws.String(os.Getenv("LANDLORD_DEFAULT_USER_POOL")),
		Schema:   customAttributes,
	}
	_, err = cogIdp.CreateUserPool(ctx, input)
	if err != nil {
		zap.L().Fatal("failed to create user pool", zap.Error(err))
	}
}
