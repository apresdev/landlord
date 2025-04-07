package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"

	"go.uber.org/zap"

	"github.com/apresdev/landlord/internal/multi"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
)

func handleListRolesForTenant(a App, c *gin.Context) error {
	sub, err := perms.SubjectFromContext(c)
	if err != nil {
		return err
	}
	t := c.Param("tenantID")
	if t == "" {
		return apperr.NewBadRequest("no tenantID")
	}
	th, err := multi.ParseTenantHandle(t)
	if err != nil {
		return apperr.NewBadRequestE(err)
	}
	ten, err := db.ResolveTenant(c.Request.Context(), a.DBCtx(), th)
	if err != nil {
		return apperr.NewNotFoundE(err)
	}
	if !perms.IsAuthorizedToReadTenant(a.RootTenantID(), ten, sub.Role) {
		zap.L().Info("user not unauthorized for tenant",
			zap.String("clientID", string(sub.Cid)),
			zap.String("username", sub.Username),
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
