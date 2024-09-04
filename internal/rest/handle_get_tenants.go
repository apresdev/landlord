package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleGetTenants(app App, c *gin.Context) error {
	ur, err := resolveUser(c)
	if err != nil {
		return err
	}
	var resp TenantsResponse
	resp.Tenants = make([]Tenant, 0)
	allTenants, err := db.GetAllTenants(c.Request.Context(), app.DDB())
	if err != nil {
		zap.L().Error("error fetching tenants", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	for _, ten := range allTenants {
		if authx.IsAuthorizedForTenant(c.Request.Context(), app.DDB(), ten, ur.Roles) {
			resp.Tenants = append(resp.Tenants, Tenant{
				Id:          ten.ID.String(),
				Name:        ten.Name,
				Description: ten.Description,
				Roles:       ten.Roles,
			})
		}
	}
	c.JSON(http.StatusOK, resp)
	return nil
}
