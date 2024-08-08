package authx

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cogidptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func SetImpersonateOnUser(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string,
	username string,
	groupName string) error {

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(username),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String("custom:impersonate"),
				Value: aws.String(groupName),
			},
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}

func ClearImpersonateOnUser(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string,
	username string) error {

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(username),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String("custom:impersonate"),
				Value: aws.String(""),
			},
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}
