package webapp

import (
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/rest"

	"github.com/gin-gonic/gin"
)

func handleUpdateMFA(a App, c *gin.Context, policy rest.MfaPolicy) error {
	tenantID := c.Param("tenantID")
	if tenantID == "" {
		return apperr.NewBadRequest("no tenant ID provided")
	}
	username := c.Param("username")
	if username == "" {
		return apperr.NewBadRequest("no user ID provided")
	}
	req := rest.UpdateUserMfaRequest{}
	req.MfaPolicy = policy
	if err := rest.HandleUpdateUserMFA(a, c, tenantID, username, req); err != nil {
		return err
	}
	user, err := db.GetUserByName(c.Request.Context(), a.DDB(), username)
	if err != nil {
		return apperr.NewInternalE(err)
	}
	render(c, userFragment(user))
	return nil
}

func handleUpdateMFAEnable(a App, c *gin.Context) error {
	return handleUpdateMFA(a, c, rest.MFAPolicyRequired)
}

func handleUpdateMFADisable(a App, c *gin.Context) error {
	return handleUpdateMFA(a, c, rest.MFAPolicyNone)
}
