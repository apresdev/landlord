package authx

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
)

func SendInvite(
	ctx context.Context,
	dbc *db.Context,
	cogIdp AWSCognitoIdentityProvider,
	ten *multi.Tenant,
	username string,
	overrideUserID uuid.UUID,
	role string,
	mfaPolicy multi.MFAPolicy) error {

	e, err := ParseEmail(username)
	if err != nil {
		return errors.New("invalid email")
	}
	if role == "" {
		return apperr.NewBadRequest("role is required")
	}
	if !ten.HasRole(role) {
		return apperr.NewBadRequest(
			fmt.Sprintf("role '%s' is not valid for tenant '%s'", role, ten.ID))
	}
	existingUser, err := db.GetUserByName(ctx, dbc, e.Address())
	if err == nil {
		if existingUser.TenantID != ten.ID {
			return apperr.NewBadRequest(
				fmt.Sprintf("user '%s' already belongs to another tenant", e.Address()))
		}
	} else if !apperr.IsNotFound(err) {
		return err
	}
	if existingUser != nil && existingUser.Status == multi.UserStatusInactive {
		return apperr.NewBadRequest(
			fmt.Sprintf("cannot send invite, user '%s' is inactive", e.Address()))
	}
	args := cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId: aws.String(ten.UserPoolID),
		Username:   aws.String(e.Address()),
	}
	// User has already been activated, so we need "resend" the invite, instead
	// of just a regular old send.
	if existingUser != nil {
		args.MessageAction = types.MessageActionTypeResend
	}
	o, err := cogIdp.AdminCreateUser(ctx, &args)
	if err != nil {
		return err
	}
	if err := SetCognitoGroup(ctx, cogIdp, ten, e.Address(), role); err != nil {
		return err
	}

	var cognitoUserID uuid.UUID
	for _, attr := range o.User.Attributes {
		if aws.ToString(attr.Name) == "sub" {
			cognitoUserID, err = uuid.Parse(aws.ToString(attr.Value))
			if err != nil {
				return err
			}
			break
		}
	}
	u := multi.NewUser(ten.ID, cognitoUserID, e.Address(), role, mfaPolicy)
	if overrideUserID != uuid.Nil {
		u.ID = overrideUserID
	}

	err = setCustomClaimsOnUser(ctx, cogIdp, ten, u)
	if err != nil {
		return err
	}

	// Since the user's do not go through the typical sign-up flow in our version
	// of Cognito, we have to manually update the email_verified attribute. We were
	// previously relying on logging into Landlord to do this, but that does not
	// happen in the typical case. There, update email_verified when invited.
	_, err = cogIdp.AdminUpdateUserAttributes(ctx,
		&cognitoidentityprovider.AdminUpdateUserAttributesInput{
			UserPoolId: aws.String(ten.UserPoolID),
			Username:   aws.String(e.Address()),
			UserAttributes: []types.AttributeType{
				{Name: aws.String("email_verified"), Value: aws.String("true")},
			},
		})
	if err != nil {
		zap.L().Error("could not update user attributes",
			zap.String("tenantID", ten.ID.String()),
			zap.String("userID", e.Address()),
			zap.String("cognitoID", e.Address()),
			zap.Error(err))
	}

	return db.StoreUser(ctx, dbc, u)
}

func DeleteCognitoUser(
	ctx context.Context,
	cogIdp AWSCognitoIdentityProvider,
	ten *multi.Tenant,
	user *multi.User) {

	args := cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: aws.String(ten.UserPoolID),
		Username:   aws.String(user.CognitoID.String()),
	}
	if _, err := cogIdp.AdminDeleteUser(ctx, &args); err != nil {
		zap.L().Error("could not delete user from Cognito",
			zap.String("tenantID", ten.ID.String()),
			zap.String("userID", user.ID.String()),
			zap.String("cognitoID", user.CognitoID.String()),
			zap.Error(err))
	}
}
