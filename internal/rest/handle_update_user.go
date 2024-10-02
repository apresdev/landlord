package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/jwx"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/apresdev/landlord/internal/authx"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

func handleUpdateUser(app App, c *gin.Context) error {
	t, ok := c.Get(TokenKey)
	if !ok {
		return apperr.NewUnauthorized("no token")
	}
	tok, ok := t.(*jwt.Token)
	if !ok {
		zap.L().Error("could not convert token to *jwt.Token")
		return apperr.NewUnauthorized("no token")
	}
	cid := clientID(c)
	groups, err := jwx.GetClaimStrings(tok, authx.CognitoGroupsClaim)
	if !ok {
		zap.L().Error("could get groups",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("token", tok))
		return apperr.NewUnauthorized("no groups")
	}
	roles, err := authx.ParseRolesFromGroups(groups)
	if err != nil {
		zap.L().Error("could not parse roles from groups",
			zap.String("clientID", string(cid)),
			zap.Error(err),
			zap.Any("groups", groups))
		return apperr.NewUnauthorized("no groups")
	}
	var resp TenantsResponse
	resp.Tenants = make([]Tenant, 0)
	allTenants, err := db.GetAllTenants(c.Request.Context(), app.DDB())
	if err != nil {
		zap.L().Error("error fetching tenants", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	for _, ten := range allTenants {
		if authx.IsAuthorizedForTenant(
			c.Request.Context(), app.DDB(), app.RootTenantID(), ten, roles) {

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
