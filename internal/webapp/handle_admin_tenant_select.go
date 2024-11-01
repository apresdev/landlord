package webapp

import (
	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/authx"
	"github.com/apresdev/landlord/internal/db"
	"github.com/apresdev/landlord/internal/multi"
	"github.com/gin-gonic/gin"
)

func handleAdminTenantSelect(a App, c *gin.Context) error {
	cu, err := resolveCurrentUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}
	action := c.Query("action")
	if action == "" {
		return apperr.NewBadRequest("missing action")
	}
	tens, err := db.GetAllTenants(c.Request.Context(), a.DDB())
	if err != nil {
		return err
	}
	rc, err := authx.RequesterFromContext(c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return err
	}
	authorizedTenants := make([]*multi.Tenant, 0)
	for _, ten := range tens {
		if authx.IsAuthorizedForTenant(a.RootTenantID(), ten, rc.Roles) {
			authorizedTenants = append(authorizedTenants, ten)
		}
	}

	render(c, tenantSelectPage("Landlord", cu, tens))
	return nil
}
