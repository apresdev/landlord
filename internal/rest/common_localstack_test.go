package rest

import (
	"github.com/apresdev/landlord/internal/appcore"
	"github.com/apresdev/landlord/internal/appenv"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/google/uuid"
)

func inviteRootUser(s appenv.Settings, username string) error {
	c := appcore.MustWithSettings(s)
	rootTenant, err := db.GetRootTenant(c.Ctx(), c.DBCtx(), c.RootTenantID())
	if err != nil {
		return err
	}
	return authx.SendInvite(
		c.Ctx(),
		c.DBCtx(),
		c.CogIdp(),
		rootTenant,
		username,
		uuid.Nil,
		"admin",
		multi.MFAPolicyNone)
}
