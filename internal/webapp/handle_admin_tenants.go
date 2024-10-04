package webapp

import (
	"github.com/apresdev/landlord/internal/db"
	"github.com/gin-gonic/gin"
)

func handleAdminTenants(a App, c *gin.Context) error {
	cu, err := resolveCurrentUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}

	// TODO: all calls to GetAllTenants and similar APIs need to check permissions.

	tens, err := db.GetAllTenants(c.Request.Context(), a.DDB())
	if err != nil {
		return err
	}
	render(c, tenantsPage("Landlord", cu, tens))
	return nil
}
