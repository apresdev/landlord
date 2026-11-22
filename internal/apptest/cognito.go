package apptest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cogtypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"go.uber.org/zap"
)

// dangerousRecreateUserPool optionally deletes the user pool with the given name
// and creates a new one with the same name.
//
// This is meant for testing only and only works against a local AWS emulator.
func dangerousRecreateUserPool(cfg aws.Config, userPoolName string) {
	ctx := context.Background()
	cogIdp := cognitoidentityprovider.NewFromConfig(cfg)
	maxResults := int32(10)
	args := cognitoidentityprovider.ListUserPoolsInput{
		MaxResults: &maxResults,
	}
	result := make([]cogtypes.UserPoolDescriptionType, 0)
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
		Required  bool
	}{
		{ClaimName: "email", MaxLen: 128, Required: true},
		{ClaimName: "tenant", MaxLen: 36},
		{ClaimName: "user", MaxLen: 36},
		{ClaimName: "impersonate_tenant", MaxLen: 36},
		{ClaimName: "impersonate_group", MaxLen: 128},
		{ClaimName: "impersonate_sub", MaxLen: 36},
		{ClaimName: "impersonate_email", MaxLen: 128},
		{ClaimName: "impersonate_user_id", MaxLen: 36},
	}
	customAttributes := make([]cogtypes.SchemaAttributeType, len(claims))
	for i, claim := range claims {
		customAttributes[i] = cogtypes.SchemaAttributeType{
			Name:                   aws.String(claim.ClaimName),
			AttributeDataType:      "String",
			DeveloperOnlyAttribute: aws.Bool(false),
			Mutable:                aws.Bool(true),
			StringAttributeConstraints: &cogtypes.StringAttributeConstraintsType{
				MinLength: aws.String("0"),
				MaxLength: aws.String(strconv.Itoa(claim.MaxLen)),
			},
			Required: aws.Bool(claim.Required),
		}
	}
	input := &cognitoidentityprovider.CreateUserPoolInput{
		PoolName:               aws.String(userPoolName),
		Schema:                 customAttributes,
		UsernameAttributes:     []cogtypes.UsernameAttributeType{"email"},
		AutoVerifiedAttributes: []cogtypes.VerifiedAttributeType{"email"},
	}
	up, err := cogIdp.CreateUserPool(ctx, input)
	if err != nil {
		zap.L().Fatal("failed to create user pool", zap.Error(err))
	}
	zap.L().Info("created user pool", zap.String("name", userPoolName))

	clientName := userPoolName + "-client"
	_, err = cogIdp.CreateUserPoolClient(ctx,
		&cognitoidentityprovider.CreateUserPoolClientInput{
			ClientName:                      aws.String(clientName),
			UserPoolId:                      up.UserPool.Id,
			AllowedOAuthFlows:               []cogtypes.OAuthFlowType{"code"},
			AllowedOAuthFlowsUserPoolClient: true,
			AllowedOAuthScopes:              []string{"email", "openid", "aws.cognito.signin.user.admin"},
			CallbackURLs:                    []string{"http://127.0.0.1:42001/callback"},
			DefaultRedirectURI:              aws.String("http://127.0.0.1:42001/"),
			ExplicitAuthFlows: []cogtypes.ExplicitAuthFlowsType{
				"ALLOW_USER_PASSWORD_AUTH", "ALLOW_REFRESH_TOKEN_AUTH", "ALLOW_USER_SRP_AUTH"},
			GenerateSecret:             true,
			LogoutURLs:                 []string{"http://127.0.0.1:42001/logout"},
			SupportedIdentityProviders: []string{"COGNITO"},
		})
	if err != nil {
		zap.L().Fatal("failed to create user pool client", zap.Error(err))
	}
	zap.L().Info("created user pool client", zap.String("name", clientName))

	lambdaFunc, err := createOrGetLambda(ctx, cfg, aws.ToString(up.UserPool.Arn))
	if err != nil {
		zap.L().Fatal("failed to create lambda function", zap.Error(err))
	}
	_, err = cogIdp.UpdateUserPool(ctx, &cognitoidentityprovider.UpdateUserPoolInput{
		UserPoolId: up.UserPool.Id,
		LambdaConfig: &cogtypes.LambdaConfigType{
			PreTokenGenerationConfig: &cogtypes.PreTokenGenerationVersionConfigType{
				LambdaArn:     lambdaFunc.Configuration.FunctionArn,
				LambdaVersion: "V2_0",
			},
		},
	})
	if err != nil {
		zap.L().Fatal("failed to update user pool", zap.Error(err))
	}
}

func createOrGetLambda(ctx context.Context, cfg aws.Config, userPoolArn string) (*lambda.GetFunctionOutput, error) {
	name := "pre-token-generation"
	lambdaClient := lambda.NewFromConfig(cfg)
	o, err := lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	})
	if err == nil {
		// The function outlives user pools, but each pool needs its own permission.
		err = addCognitoPermission(ctx, lambdaClient, *o.Configuration.FunctionArn, userPoolArn)
		if err != nil {
			return nil, err
		}
		return o, nil
	}

	environment := "TestEnvironment"
	gitDir, err := findGitDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to find git directory: %w", err)
	}
	sourceFile := filepath.Join(gitDir, "internal/apptest/landlord_pre_token_generation_lambda.mjs.zip")
	runtime := "nodejs20.x"
	handler := "landlord_pre_token_generation_lambda.handler"

	lambdaFunction, err := createLambdaFunction(
		ctx,
		lambdaClient,
		name,
		environment,
		sourceFile,
		runtime,
		handler,
	)
	if err != nil {
		return nil, err
	}

	err = addCognitoPermission(
		ctx,
		lambdaClient,
		*lambdaFunction.Configuration.FunctionArn,
		userPoolArn,
	)
	if err != nil {
		return nil, err
	}

	return lambdaFunction, nil
}

func createLambdaFunction(
	ctx context.Context,
	lambdaClient *lambda.Client,
	name string,
	environment string,
	sourceFile string,
	runtime string,
	handler string,
) (*lambda.GetFunctionOutput, error) {
	// A fake role ARN that the emulator accepts
	roleArn := fmt.Sprintf("arn:aws:iam::123456789012:role/%s-execution-role", name)

	zipBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read lambda source file: %w", err)
	}

	_, err = lambdaClient.CreateFunction(ctx, &lambda.CreateFunctionInput{
		FunctionName: aws.String(name),
		Role:         aws.String(roleArn),
		Code: &lambdatypes.FunctionCode{
			ZipFile: zipBytes,
		},
		Runtime: lambdatypes.Runtime(runtime),
		Handler: aws.String(handler),
		Environment: &lambdatypes.Environment{
			Variables: map[string]string{
				"ENVIRONMENT": environment,
			},
		},
		Tags: map[string]string{
			"Name":        name,
			"Environment": environment,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create lambda function: %w", err)
	}

	return lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(name),
	})
}

func addCognitoPermission(
	ctx context.Context,
	lambdaClient *lambda.Client,
	functionArn string,
	cognitoUserPoolArn string,
) error {
	_, err := lambdaClient.AddPermission(ctx, &lambda.AddPermissionInput{
		Action:       aws.String("lambda:InvokeFunction"),
		FunctionName: aws.String(functionArn),
		Principal:    aws.String("cognito-idp.amazonaws.com"),
		SourceArn:    aws.String(cognitoUserPoolArn),
		StatementId:  aws.String("AllowExecuteFromCognito-" + sourceArnSuffix(cognitoUserPoolArn)),
	})

	return err
}

// sourceArnSuffix returns the user pool ID part of a user pool ARN, for use in
// a unique policy statement ID.
func sourceArnSuffix(arn string) string {
	return arn[strings.LastIndex(arn, "/")+1:]
}
