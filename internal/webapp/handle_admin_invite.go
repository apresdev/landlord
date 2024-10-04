package webapp

import (
	"fmt"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/db"
	"github.com/gin-gonic/gin"
)

func handleAdminInvite(a App, c *gin.Context) error {
	cUser, err := resolveClientUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}
	tenantID := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(tenantID)
	if err != nil {
		return apperr.NewBadRequestE(
			fmt.Errorf("error parsing tenant ID: %w", err))
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil && !apperr.IsNotFound(err) {
		return apperr.NewBadRequestE(
			fmt.Errorf("error resolving tenant: %w", err))
	}
	// Okay for the `username` to be empty. Simply provides a default value
	// for the UI in situations where we navigate from a place with a known
	// username.
	usernameToInvite := c.Query("username")
	render(c, inviteUserPage(
		a.Settings().AppName(),
		ten,
		usernameToInvite,
		ten.Roles,
		multi.MFAPolicyStrings(),
		cUser.User))
	return nil

}
