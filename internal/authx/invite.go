package authx

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
)

func SendInvite(
	ctx context.Context,
	ddb db.AWSDynamoDB,
	cogIdp AWSCognitoIdentityProvider,
	ten *multi.Tenant,
	username string,
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
	existingUser, err := db.GetUserByName(ctx, ddb, e.Address())
	if err == nil {
		if existingUser.TenantID != ten.ID {
			return apperr.NewBadRequest(
				fmt.Sprintf("user '%s' already belongs to another tenant", e.Address()))
		}
	} else if err != nil && !apperr.IsNotFound(err) {
		return err
	}
	if existingUser != nil && existingUser.Status == multi.UserStatusInactive {
		return apperr.NewBadRequest(
			fmt.Sprintf("cannot send invite, user '%s' is inactive", e.Address()))
	}
	ua := []types.AttributeType{
		{Name: aws.String(CustomTenantClaim), Value: aws.String(ten.ID.String())},
	}
	args := cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId:     aws.String(ten.UserPoolID),
		Username:       aws.String(e.Address()),
		UserAttributes: ua,
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
	return db.StoreUser(
		ctx, ddb, multi.NewUser(ten.ID, cognitoUserID, e.Address(), role, mfaPolicy))
}

func CancelInvitation(
	ctx context.Context,
	ddb db.AWSDynamoDB,
	cogIdp AWSCognitoIdentityProvider,
	username string) error {

	e, err := ParseEmail(username)
	if err != nil {
		return errors.New("invalid email")
	}
	u, ten, err := db.GetUserAndTenant(ctx, ddb, e.Address())
	if err != nil {
		return err
	}
	if u.Status != multi.UserStatusInvited {
		return apperr.NewBadRequest(
			fmt.Sprintf("cannot cancel invite, user '%s' is not invited", e.Address()))
	}
	args := cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: aws.String(ten.UserPoolID),
		Username:   aws.String(e.Address()),
	}
	if _, err := cogIdp.AdminDeleteUser(ctx, &args); err != nil {
		zap.L().Error("could not delete user from Cognito",
			zap.String("username", e.Address()),
			zap.Error(err))
	}
	return db.DeleteUser(ctx, ddb, u)
}
