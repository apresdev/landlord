package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func convertTenant(ten *multi.Tenant) Tenant {
	return Tenant{
		Id:             ten.ID.String(),
		Name:           ten.Name,
		Description:    ten.Description,
		Roles:          ten.Roles,
		SubTenantRoles: ten.SubTenantRoles,
		ProfileFields:  convertProfileFields(ten),
	}
}

func handleListTenants(a App, c *gin.Context) error {
	tens, err := HandleGetTenants(a, c)
	if err != nil {
		return err
	}
	var resp TenantsResponse
	resp.Tenants = make([]Tenant, len(tens))
	for i, ten := range tens {
		resp.Tenants[i] = convertTenant(ten)
	}
	c.JSON(http.StatusOK, resp)
	return nil
}

func HandleGetTenants(a App, c *gin.Context) ([]*multi.Tenant, error) {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return nil, err
	}
	allTenants, err := db.GetAllTenants(c.Request.Context(), a.DBCtx())
	if err != nil {
		zap.L().Error("error fetching tenants", zap.Error(err))
		return nil, apperr.NewUnauthorized("not authorized")
	}
	result := make([]*multi.Tenant, 0, len(allTenants))
	for _, ten := range allTenants {
		if perms.IsAuthorizedToReadTenant(a.RootTenantID(), ten, sub.Role) {
			result = append(result, ten)
		}
	}
	return result, nil
}
