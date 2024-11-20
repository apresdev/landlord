package rest

import (
	"net/http"

	"github.com/apresdev/landlord/internal/perms"

	"github.com/apresdev/landlord/internal/apperr"

	"github.com/apresdev/landlord/internal/db"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func handleUpdateUserRole(app App, c *gin.Context) error {
	p, err := perms.PrincipalFromContext(c)
	if err != nil {
		return apperr.NewUnauthorized("no token")
	}
	var req UpdateUserRoleRequest
	if err := c.BindJSON(&req); err != nil {
		return apperr.NewBadRequest(err.Error())
	}
	tenantID, err := UUIDParam(c, "tenantID")
	if err != nil {
		return apperr.NewBadRequest("invalid tenant ID")
	}
	userID, err := UUIDParam(c, "userID")
	if err != nil {
		return apperr.NewBadRequest("invalid user ID")
	}
	targetTenant, err := db.GetTenantByID(c.Request.Context(), app.DDB(), tenantID)
	if err != nil {
		zap.L().Error("error fetching tenant",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()))
		return apperr.NewUnauthorized("not authorized")
	}
	targetUser, err := db.GetUserInTenantByID(c.Request.Context(), app.DDB(), tenantID, userID)
	if err != nil {
		zap.L().Error("error fetching user",
			zap.Error(err),
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()))
		return apperr.NewUnauthorized("not authorized")
	}

	resource := perms.NewResource(
		perms.ResourceKindUserRole, req.Role, targetTenant, targetUser)
	effect, err := perms.Eval(c, app, p, perms.OperationWrite, resource)
	if err != nil {
		zap.L().Error("could not evaluate permissions", zap.Error(err))
		return apperr.NewUnauthorized("not authorized")
	}
	if effect.IsDenied() {
		zap.L().Info("denied setting role for user",
			zap.String("principal", p.String()),
			zap.String("resource", resource.String()))
		return apperr.NewUnauthorized("not authorized")
	}

	targetUser.Role = req.Role
	err = db.StoreUser(c.Request.Context(), app.DDB(), targetUser)
	if err != nil {
		zap.L().Error("error updating user",
			zap.String("tenantID", tenantID.String()),
			zap.String("userID", userID.String()),
			zap.Error(err))
		return apperr.NewInternal("error updating user")
	}

	c.JSON(http.StatusNoContent, nil)
	return nil
}
