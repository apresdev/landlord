package webapp

import (
	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
)

func handleDeactivateUser(a App, c *gin.Context) error {
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenant ID provided")
	}
	username := c.Param("username")
	if username == "" {
		return apperr.NewBadRequest("no username provided")
	}
	user, err := rest.HandleSetUserStatus(
		a, c, tenantID, username, string(multi.TenantStateInactive))
	if err != nil {
		return err
	}
	render(c, userFragment(user))
	return nil
}
