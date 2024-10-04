package webapp

import (
	"fmt"
	"sort"

	"github.com/apresdev/landlord/internal/apperr"
	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/db"
	"github.com/gin-gonic/gin"
)

func handleAdminTenant(a App, c *gin.Context) error {
	cu, err := resolveCurrentUser(a, c)
	if err != nil {
		clearLandlordCookies(c, a.Settings().URL())
		redirectToLogin(c)
		return nil
	}

	id := c.Param("tenantID")
	th, err := multi.ParseTenantHandle(id)
	if err != nil {
		return apperr.NewBadRequestE(
			fmt.Errorf("error parsing tenant ID: %w", err))
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DDB(), th)
	if err != nil && !apperr.IsNotFound(err) {
		return apperr.NewNotFoundE(
			fmt.Errorf("error resolving tenant: %w", err))
	}
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting tenants: %w", err))
	}

	// TODO: handle no users for tenant more gracefully

	users, err := db.GetUsersForTenant(c.Request.Context(), a.DDB(), ten.ID)
	if err != nil {
		return apperr.NewInternalE(
			fmt.Errorf("error getting users for tenant: %w", err))
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Username < users[j].Username
	})
	render(c, tenantPage(a.Settings().AppName(), cu, ten, users))
	return nil
}
