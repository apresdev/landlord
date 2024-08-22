package authx

import (
	"context"
	"errors"
	"fmt"

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
	u, err := db.GetUser(ctx, ddb, e.Address())
	if err == nil {
		if u.Status == multi.UserStatusInactive {
			return apperr.NewBadRequest(
				fmt.Sprintf("cannot send invite, user '%s' is inactive", e.Address()))
		}
	} else if !apperr.IsNotFound(err) {
		return err
	}
	ua := []types.AttributeType{
		{Name: aws.String(TenantClaim), Value: aws.String(ten.ID.String())},
	}
	args := cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId:     aws.String(ten.UserPoolID),
		Username:       aws.String(e.Address()),
		UserAttributes: ua,
	}
	// User has already been activated, so we need to resend the invite.
	if u != nil {
		args.MessageAction = types.MessageActionTypeResend
	}
	if _, err := cogIdp.AdminCreateUser(ctx, &args); err != nil {
		return err
	}
	return db.StoreUser(ctx, ddb, &multi.User{
		Username:  e.Address(),
		TenantID:  ten.ID,
		Role:      role,
		Status:    multi.UserStatusInvited,
		MFAPolicy: mfaPolicy,
	})
}
