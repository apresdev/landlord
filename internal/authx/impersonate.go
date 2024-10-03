package authx

import (
	"context"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cogidptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

func ImpersonateRole(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	userPoolID string,
	subjectUsername string,
	impersonateTenant uuid.UUID,
	impersonateGroup string) error {

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(subjectUsername),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String(CustomImpersonateTenantClaim),
				Value: aws.String(impersonateTenant.String()),
			},
			{
				Name:  aws.String(CustomImpersonateGroupClaim),
				Value: aws.String(impersonateGroup),
			},
			{
				Name:  aws.String(CustomImpersonateEmailClaim),
				Value: aws.String(""),
			},
			{
				Name:  aws.String(CustomImpersonateSubClaim),
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
	subjectUsername string,
	impersonateTenant uuid.UUID,
	impersonateGroup string,
	impersonateEmail string,
	impersonateSub uuid.UUID) error {

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(subjectUsername),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String(CustomImpersonateTenantClaim),
				Value: aws.String(impersonateTenant.String()),
			},
			{
				Name:  aws.String(CustomImpersonateGroupClaim),
				Value: aws.String(impersonateGroup),
			},
			{
				Name:  aws.String(CustomImpersonateEmailClaim),
				Value: aws.String(impersonateEmail),
			},
			{
				Name:  aws.String(CustomImpersonateSubClaim),
				Value: aws.String(impersonateSub.String()),
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
	subjectUsername string) error {

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(userPoolID),
		Username:   aws.String(subjectUsername),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String(CustomImpersonateTenantClaim),
				Value: aws.String(""),
			},
			{
				Name:  aws.String(CustomImpersonateGroupClaim),
				Value: aws.String(""),
			},
			{
				Name:  aws.String(CustomImpersonateEmailClaim),
				Value: aws.String(""),
			},
			{
				Name:  aws.String(CustomImpersonateSubClaim),
				Value: aws.String(""),
			},
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}
