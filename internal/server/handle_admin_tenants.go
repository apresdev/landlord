package server

import (
	"net/http"

	"github.com/apresdev/landlord/internal/db"
	"github.com/gin-gonic/gin"
)

func handleAdminTenants(a *app, c *gin.Context) error {
	cUser, err := resolveClientUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		c.Redirect(http.StatusFound, "/login")
		return nil
	}

	// TODO: all calls to GetAllTenants and similar APIs need to check permissions.

	tens, err := db.GetAllTenants(c.Request.Context(), a.DDB())
	if err != nil {
		return err
	}
	render(c, tenantsPage("Landlord", tens, cUser.User))
	return nil
}
