package authx

import (
	"context"

	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
)

func OnSuccessfulLogin(
	ctx context.Context,
	dbc *db.Context,
	user *multi.User) error {

	if user.Status == multi.UserStatusInvited {
		user.Status = multi.UserStatusActive
		if err := db.StoreUser(ctx, dbc, user); err != nil {
			return err
		}
	}
	return nil
}
