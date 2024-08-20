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

type Resend int

const (
	NoResend Resend = iota
	ResendConfirmation
)

func SendInvite(
	ctx context.Context,
	ddb db.AWSDynamoDB,
	cogIdp AWSCognitoIdentityProvider,
	ten *multi.Tenant,
	username string,
	role string,
	resend Resend) error {

	e, err := ParseEmail(username)
	if err != nil {
		return errors.New("invalid email")
	}
	_, err = db.GetUser(ctx, ddb, e.Address())
	if err == nil {
		return apperr.NewAlreadyExists(
			fmt.Sprintf("%s is already signed up", e.Address()))
	}
	ua := []types.AttributeType{
		{Name: aws.String("custom:tenant"), Value: aws.String(ten.ID.String())},
	}
	args := cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId:     aws.String(ten.UserPoolID),
		Username:       aws.String(e.Address()),
		UserAttributes: ua,
	}
	if resend == ResendConfirmation {
		args.MessageAction = types.MessageActionTypeResend
	}
	if _, err := cogIdp.AdminCreateUser(ctx, &args); err != nil {
		return err
	}
	return db.StoreUser(ctx, ddb, &multi.User{
		Username: e.Address(),
		TenantID: ten.ID,
		Role:     role,
		Status:   multi.UserStatusInvited,
	})
}
