package authx

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cogidptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func ImpersonateRole(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string,
	username string,
	roleName string) error {

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(username),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String(ImpersonateGroupClaim),
				Value: aws.String(roleName),
			},
			{
				Name:  aws.String(ImpersonateUserClaim),
				Value: aws.String(""),
			},
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}

func ImpersonateUser(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string,
	username string,
	impersonateUsername string) error {

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(username),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String(ImpersonateGroupClaim),
				Value: aws.String(""),
			},
			{
				Name:  aws.String(ImpersonateUserClaim),
				Value: aws.String(impersonateUsername),
			},
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}

func ClearImpersonate(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string,
	username string) error {

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(username),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String(ImpersonateGroupClaim),
				Value: aws.String(""),
			},
			{
				Name:  aws.String(ImpersonateUserClaim),
				Value: aws.String(""),
			},
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}
