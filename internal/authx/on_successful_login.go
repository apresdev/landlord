package authx

import (
	"context"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
)

func OnSuccessfulLogin(
	ctx context.Context,
	ddb db.AWSDynamoDB,
	cogIdp AWSCognitoIdentityProvider,
	tenant *multi.Tenant,
	user *multi.User) error {

	if user.Status == multi.UserStatusInvited {
		user.Status = multi.UserStatusActive
		if err := db.StoreUser(ctx, ddb, user); err != nil {
			return err
		}
		if err := SetCognitoGroup(ctx, cogIdp, tenant, user); err != nil {
			return err
		}
	}
	return nil
}
