package authx

import (
	"context"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cogidptypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
)

// ImpersonateRole updates the impersonation attributes of a user in
// the target tenant's user pool.
func ImpersonateRole(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	subjectUsername string,
	targetTenant *multi.Tenant,
	impersonateGroup string) error {

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(targetTenant.UserPoolID),
		Username:   aws.String(subjectUsername),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String(CustomImpersonateTenantClaim),
				Value: aws.String(targetTenant.ID.String()),
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
			{
				Name:  aws.String(CustomImpersonateUserClaim),
				Value: aws.String(""),
			},
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}

// ImpersonateUser updates the impersonation attributes of a user in
// the target tenant's user pool.
func ImpersonateUser(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	subjectUsername string,
	targetTenant *multi.Tenant,
	targetUser *multi.User) error {

	targetGroupName := targetTenant.GroupName(targetUser.Role)
	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId: aws.String(targetTenant.UserPoolID),
		Username:   aws.String(subjectUsername),
		UserAttributes: []cogidptypes.AttributeType{
			{
				Name:  aws.String(CustomImpersonateTenantClaim),
				Value: aws.String(targetTenant.ID.String()),
			},
			{
				Name:  aws.String(CustomImpersonateGroupClaim),
				Value: aws.String(targetGroupName),
			},
			{
				Name:  aws.String(CustomImpersonateEmailClaim),
				Value: aws.String(targetUser.Username),
			},
			{
				Name:  aws.String(CustomImpersonateSubClaim),
				Value: aws.String(targetUser.CognitoID.String()),
			},
			{
				Name:  aws.String(CustomImpersonateUserClaim),
				Value: aws.String(targetUser.ID.String()),
			},
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}

// ClearImpersonate clears the impersonation attributes of a user in
// the target tenant's user pool.
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
			{
				Name:  aws.String(CustomImpersonateUserClaim),
				Value: aws.String(""),
			},
		},
	}
	_, err := cogIdp.AdminUpdateUserAttributes(ctx, input)
	return err
}
