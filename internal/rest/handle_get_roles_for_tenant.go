package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/authx"
	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
)

func handleGetRolesForTenant(app App, c *gin.Context) error {
	rc, err := authx.RequesterFromContext(c)
	if err != nil {
		return err
	}
	t := c.Param("tenantID")
	if t == "" {
		return apperr.NewBadRequest("no tenantID")
	}
	th, err := multi.ParseTenantHandle(t)
	if err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	ten, err := db.ResolveTenant(c.Request.Context(), app.DDB(), th)
	if err != nil {
		return apperr.NewNotFound(err.Error())
	}
	if !authx.IsAuthorizedForTenant(app.RootTenantID(), ten, rc.Roles) {

		zap.L().Info("user not unauthorized for tenant",
			zap.String("clientID", string(rc.Cid)),
			zap.String("username", rc.Username),
			zap.String("tenantID", ten.ID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	resp := TenantRolesResponse{
		Roles:          ten.Roles,
		SubTenantRoles: ten.SubTenantRoles,
	}
	c.JSON(http.StatusOK, resp)
	return nil
}
